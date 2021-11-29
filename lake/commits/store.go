package commits

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zngbytes"
	"github.com/segmentio/ksuid"
)

var (
	ErrBadCommitObject = errors.New("first record of object not a commit")
	ErrExists          = errors.New("commit object already exists")
	ErrNotFound        = errors.New("commit object not found")
)

type Store struct {
	path   *storage.URI
	engine storage.Engine
	// We use these poor-man's caches for now since chasing linked lists
	// over cloud storage will run very slowly... there is an interesting
	// scaling problem to work on here.
	cache     map[ksuid.KSUID]*Object
	snapshots map[ksuid.KSUID]*Snapshot
	paths     map[ksuid.KSUID][]ksuid.KSUID
}

func OpenStore(engine storage.Engine, path *storage.URI) (*Store, error) {
	return &Store{
		path:      path,
		engine:    engine,
		cache:     make(map[ksuid.KSUID]*Object),
		snapshots: make(map[ksuid.KSUID]*Snapshot),
		paths:     make(map[ksuid.KSUID][]ksuid.KSUID),
	}, nil
}

func (s *Store) Get(ctx context.Context, commit ksuid.KSUID) (*Object, error) {
	if o, ok := s.cache[commit]; ok {
		return o, nil
	}
	r, err := s.engine.Get(ctx, s.pathOf(commit))
	if err != nil {
		return nil, err
	}
	o, err := DecodeObject(r)
	if err == ErrBadCommitObject {
		err = fmt.Errorf("system error: %s: %w", s.pathOf(commit), ErrBadCommitObject)
	}
	if closeErr := r.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, err
	}
	//XXX do a length check on the table to limit memory use
	s.cache[commit] = o
	return o, nil
}

func (s *Store) pathOf(commit ksuid.KSUID) *storage.URI {
	return s.path.AppendPath(commit.String() + ".zng")
}

func (s *Store) Put(ctx context.Context, o *Object) error {
	b, err := o.Serialize()
	if err != nil {
		return err
	}
	return storage.Put(ctx, s.engine, s.pathOf(o.Commit), bytes.NewReader(b))
}

// DANGER ZONE - objects should only be removed when GC says they are not used.
func (s *Store) Remove(ctx context.Context, o *Object) error {
	return s.engine.Delete(ctx, s.pathOf(o.Commit))
}

func (s *Store) Snapshot(ctx context.Context, leaf ksuid.KSUID) (*Snapshot, error) {
	if snap, ok := s.snapshots[leaf]; ok {
		return snap, nil
	}
	if snap, err := s.getSnapshot(ctx, leaf); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	} else if err == nil {
		s.snapshots[leaf] = snap
		return snap, nil
	}
	var objects []*Object
	var base *Snapshot
	for at := leaf; at != ksuid.Nil; {
		if snap, ok := s.snapshots[at]; ok {
			base = snap
			break
		}
		o, err := s.Get(ctx, at)
		if err != nil {
			return nil, err
		}
		objects = append(objects, o)
		at = o.Parent
	}
	var snap *Snapshot
	if base == nil {
		snap = NewSnapshot()
	} else {
		snap = base.Copy()
	}
	for k := len(objects) - 1; k >= 0; k-- {
		for _, action := range objects[k].Actions {
			if err := PlayAction(snap, action); err != nil {
				return nil, err
			}
		}
	}
	if err := s.putSnapshot(ctx, leaf, snap); err != nil {
		return nil, err
	}
	s.snapshots[leaf] = snap
	return snap, nil
}

func (s *Store) getSnapshot(ctx context.Context, commit ksuid.KSUID) (*Snapshot, error) {
	r, err := s.engine.Get(ctx, s.snapshotPathOf(commit))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return decodeSnapshot(r)
}

func (s *Store) putSnapshot(ctx context.Context, commit ksuid.KSUID, snap *Snapshot) error {
	b, err := snap.serialize()
	if err != nil {
		return err
	}
	return storage.Put(ctx, s.engine, s.snapshotPathOf(commit), bytes.NewReader(b))
}

func (s *Store) snapshotPathOf(commit ksuid.KSUID) *storage.URI {
	return s.path.AppendPath(commit.String() + ".snap.zng")
}

// Path return the entire path from the commit object to the root
// in leaf to root order.
func (s *Store) Path(ctx context.Context, leaf ksuid.KSUID) ([]ksuid.KSUID, error) {
	if leaf == ksuid.Nil {
		return nil, errors.New("no path for nil commit ID")
	}
	if path, ok := s.paths[leaf]; ok {
		return path, nil
	}
	path, err := s.PathRange(ctx, leaf, ksuid.Nil)
	if err != nil {
		return nil, err
	}
	s.paths[leaf] = path
	return path, nil
}

func (s *Store) PathRange(ctx context.Context, from, to ksuid.KSUID) ([]ksuid.KSUID, error) {
	var path []ksuid.KSUID
	for at := from; at != ksuid.Nil; {
		if cache, ok := s.paths[at]; ok {
			for _, id := range cache {
				path = append(path, id)
				if id == to {
					break
				}
			}
			break
		}
		path = append(path, at)
		o, err := s.Get(ctx, at)
		if err != nil {
			return nil, err
		}
		if at == to {
			break
		}
		at = o.Parent
	}
	return path, nil
}

func (s *Store) GetBytes(ctx context.Context, commit ksuid.KSUID) ([]byte, *Commit, error) {
	b, err := storage.Get(ctx, s.engine, s.pathOf(commit))
	if err != nil {
		return nil, nil, err
	}
	reader := zngbytes.NewDeserializer(bytes.NewReader(b), ActionTypes)
	entry, err := reader.Read()
	if err != nil {
		return nil, nil, err
	}
	first, ok := entry.(*Commit)
	if !ok {
		return nil, nil, fmt.Errorf("system error: first record of commit object is not a commit action: %s", s.pathOf(commit))
	}
	return b, first, nil
}

func (s *Store) ReadAll(ctx context.Context, commit, stop ksuid.KSUID) ([]byte, error) {
	var size int
	var buffers [][]byte
	for commit != ksuid.Nil && commit != stop {
		b, commitObject, err := s.GetBytes(ctx, commit)
		if err != nil {
			return nil, err
		}
		size += len(b)
		buffers = append(buffers, b)
		commit = commitObject.Parent
	}
	out := make([]byte, 0, size)
	for k := len(buffers) - 1; k >= 0; k-- {
		out = append(out, buffers[k]...)
	}
	return out, nil
}

func (s *Store) Open(ctx context.Context, commit, stop ksuid.KSUID) (io.Reader, error) {
	b, err := s.ReadAll(ctx, commit, stop)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

func (s *Store) OpenAsZNG(ctx context.Context, zctx *zed.Context, commit, stop ksuid.KSUID) (*zngio.Reader, error) {
	r, err := s.Open(ctx, commit, stop)
	if err != nil {
		return nil, err
	}
	return zngio.NewReader(r, zctx), nil
}

func (s *Store) OpenCommitLog(ctx context.Context, zctx *zed.Context, commit, stop ksuid.KSUID) zio.Reader {
	return newLogReader(ctx, zctx, s, commit, stop)
}

// PatchOfCommit computes the snapshot at the parent of the indicated commit
// then computes the difference between that snapshot and the child commit,
// returning the difference as a patch.
func (s *Store) PatchOfCommit(ctx context.Context, commit ksuid.KSUID) (*Patch, error) {
	path, err := s.Path(ctx, commit)
	if err != nil {
		return nil, err
	}
	if len(path) == 0 {
		return nil, errors.New("system error: no error on pathless commit")
	}
	var base *Snapshot
	if len(path) == 1 {
		// For first commit in branch, just create an empty base ...
		base = NewSnapshot()
	} else {
		parent := path[1]
		base, err = s.Snapshot(ctx, parent)
		if err != nil {
			return nil, err
		}
	}
	patch := NewPatch(base)
	object, err := s.Get(ctx, commit)
	if err != nil {
		return nil, err
	}
	for _, action := range object.Actions {
		if err := PlayAction(patch, action); err != nil {
			return nil, err
		}
	}
	return patch, nil
}

func (s *Store) PatchOfPath(ctx context.Context, base *Snapshot, baseID, commit ksuid.KSUID) (*Patch, error) {
	path, err := s.PathRange(ctx, commit, baseID)
	if err != nil {
		return nil, err
	}
	patch := NewPatch(base)
	if len(path) < 2 {
		// There are no changes past the base.  Return the empty patch.
		return patch, nil
	}
	// Play objects in forward order skipping over the last path element
	// as that is the base and the difference is relative to it.
	for k := len(path) - 2; k >= 0; k-- {
		o, err := s.Get(ctx, path[k])
		if err != nil {
			return nil, err
		}
		for _, action := range o.Actions {
			if err := PlayAction(patch, action); err != nil {
				return nil, err
			}
		}
	}
	return patch, nil
}

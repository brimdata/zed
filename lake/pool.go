package lake

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/lake/commit"
	"github.com/brimdata/zed/lake/commit/actions"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zqe"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

const (
	DataTag  = "D"
	IndexTag = "I"
	LogTag   = "L"
	StageTag = "S"
)

type PoolConfig struct {
	Version  int            `zng:"version"`
	Name     string         `zng:"name"`
	ID       ksuid.KSUID    `zng:"id"`
	Keys     []field.Static `zng:"keys"`
	Order    zbuf.Order     `zng:"order"`
	Treshold int64          `zng:"threshold"`
}

type Pool struct {
	PoolConfig
	Path      iosrc.URI
	DataPath  iosrc.URI
	IndexPath iosrc.URI
	StagePath iosrc.URI
	log       *commit.Log
}

func NewPoolConfig(name string, id ksuid.KSUID, keys []field.Static, order zbuf.Order, thresh int64) *PoolConfig {
	if thresh == 0 {
		thresh = segment.DefaultThreshold
	}
	return &PoolConfig{
		Version:  0,
		Name:     name,
		ID:       id,
		Keys:     keys,
		Order:    order,
		Treshold: thresh,
	}
}

func (p *PoolConfig) Path(root iosrc.URI) iosrc.URI {
	return root.AppendPath(p.ID.String())
}

func (p *PoolConfig) Create(ctx context.Context, root iosrc.URI) error {
	path := p.Path(root)
	if err := iosrc.MkdirAll(DataPath(path), 0700); err != nil {
		return err
	}
	if err := iosrc.MkdirAll(IndexPath(path), 0700); err != nil {
		return err
	}
	if err := iosrc.MkdirAll(StagePath(path), 0700); err != nil {
		return err
	}
	_, err := commit.Create(ctx, LogPath(path), p.Order)
	return err
}

func (p *PoolConfig) Open(ctx context.Context, root iosrc.URI) (*Pool, error) {
	path := p.Path(root)
	log, err := commit.Open(ctx, path.AppendPath(LogTag), p.Order)
	if err != nil {
		return nil, err
	}
	pool := &Pool{
		PoolConfig: *p,
		Path:       path,
		DataPath:   DataPath(path),
		IndexPath:  IndexPath(path),
		StagePath:  StagePath(path),
		log:        log,
	}
	return pool, nil
}

func (p *PoolConfig) Delete(ctx context.Context, root iosrc.URI) error {
	return iosrc.RemoveAll(ctx, p.Path(root))
}

func (p *Pool) Add(ctx context.Context, r zbuf.Reader) (ksuid.KSUID, error) {
	w, err := NewWriter(ctx, p)
	if err != nil {
		return ksuid.Nil, err
	}
	err = zbuf.CopyWithContext(ctx, w, r)
	if closeErr := w.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return ksuid.Nil, err
	}
	id := ksuid.New()
	txn := commit.NewAddsTxn(id, w.Segments())
	if err := p.StoreInStaging(ctx, txn); err != nil {
		return ksuid.Nil, err
	}
	return id, nil
}

func (p *Pool) Delete(ctx context.Context, ids []ksuid.KSUID) (ksuid.KSUID, error) {
	id := ksuid.New()
	// IDs aren't vetted here and will fail at commit time if problematic.
	txn := commit.NewDeletesTxn(id, ids)
	if err := p.StoreInStaging(ctx, txn); err != nil {
		return ksuid.Nil, err
	}
	return id, nil
}

func (p *Pool) Commit(ctx context.Context, id ksuid.KSUID, date nano.Ts, author, message string) error {
	if date == 0 {
		date = nano.Now()
	}
	txn, err := p.LoadFromStaging(ctx, id)
	if err != nil {
		if !zqe.IsNotFound(err) {
			return err
		}
		return fmt.Errorf("commit ID not staged: %s", id)
	}
	txn.AppendCommitMessage(id, date, author, message)
	if err := p.log.Commit(ctx, txn); err != nil {
		return err
	}
	// Commit succeeded.  Delete the staging entry.
	return p.ClearFromStaging(ctx, id)
}

func (p *Pool) Squash(ctx context.Context, ids []ksuid.KSUID, date nano.Ts, author, message string) (ksuid.KSUID, error) {
	if date == 0 {
		date = nano.Now()
	}
	head, err := p.log.Head(ctx)
	if err != nil {
		if err != journal.ErrEmpty {
			return ksuid.Nil, err
		}
		head = commit.NewSnapshot()
	}
	patch := commit.NewPatch(head)
	for _, id := range ids {
		txn, err := p.LoadFromStaging(ctx, id)
		if err != nil {
			if !zqe.IsNotFound(err) {
				return ksuid.Nil, err
			}
			return ksuid.Nil, fmt.Errorf("commit ID not staged: %s", id)
		}
		commit.Play(patch, txn)
	}
	txn := patch.NewTransaction()
	if err := p.StoreInStaging(ctx, txn); err != nil {
		return ksuid.KSUID{}, err
	}
	return txn.ID, nil
}

func (p *Pool) LookupTags(ctx context.Context, tags []ksuid.KSUID) ([]ksuid.KSUID, error) {
	var ids []ksuid.KSUID
	for _, tag := range tags {
		ok, err := p.SegmentExists(ctx, tag)
		if err != nil {
			return nil, err
		}
		if ok {
			ids = append(ids, tag)
			continue
		}
		snap, ok, err := p.log.SnapshotOfCommit(ctx, 0, tag)
		if err != nil {
			return nil, fmt.Errorf("tag does not exist: %s", tag)
		}
		if !ok {
			return nil, fmt.Errorf("commit tag was previously deleted: %s", tag)
		}
		for _, seg := range snap.SelectAll() {
			ids = append(ids, seg.ID)
		}
	}
	return ids, nil
}

func (p *Pool) SegmentExists(ctx context.Context, id ksuid.KSUID) (bool, error) {
	return iosrc.Exists(ctx, segment.RowObjectPath(p.DataPath, id))
}

func (p *Pool) ClearFromStaging(ctx context.Context, id ksuid.KSUID) error {
	return iosrc.Remove(ctx, p.StagingObject(id))
}

func (p *Pool) ListStagedCommits(ctx context.Context) ([]ksuid.KSUID, error) {
	infos, err := iosrc.ReadDir(ctx, p.StagePath)
	if err != nil {
		return nil, err
	}
	ids := make([]ksuid.KSUID, 0, len(infos))
	for _, info := range infos {
		_, name := filepath.Split(info.Name())
		base := strings.TrimSuffix(name, ".zng")
		id, err := ksuid.Parse(base)
		if err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (p *Pool) StagingObject(id ksuid.KSUID) iosrc.URI {
	return p.StagePath.AppendPath(id.String() + ".zng")
}

func (p *Pool) Log() *commit.Log {
	return p.log
}

func (p *Pool) LoadFromStaging(ctx context.Context, id ksuid.KSUID) (*commit.Transaction, error) {
	return commit.LoadTransaction(ctx, id, p.StagingObject(id))
}

func (p *Pool) StoreInStaging(ctx context.Context, txn *commit.Transaction) error {
	b, err := txn.Serialize()
	if err != nil {
		return fmt.Errorf("pool %q: internal error: serialize transaction: %w", p.Name, err)
	}
	return iosrc.WriteFile(ctx, p.StagingObject(txn.ID), b)
}

func (p *Pool) ScanStaging(ctx context.Context, w zbuf.Writer, ids []ksuid.KSUID) error {
	ch := make(chan actions.Interface, 10)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var err error
	go func() {
		defer close(ch)
		for _, id := range ids {
			var txn *commit.Transaction
			txn, err = p.LoadFromStaging(ctx, id)
			if err != nil {
				return
			}
			for _, action := range txn.Actions {
				select {
				case ch <- action:
				case <-ctx.Done():
					return
				}
			}
			select {
			case ch <- &actions.StagedCommit{Commit: id}:
			case <-ctx.Done():
				return
			}
		}
	}()
	m := zson.NewZNGMarshaler()
	m.Decorate(zson.StyleSimple)
	for p := range ch {
		rec, err := m.MarshalRecord(p)
		if err != nil {
			return err
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	return err
}

func (p *Pool) Scan(ctx context.Context, snap *commit.Snapshot, ch chan segment.Reference) error {
	for _, seg := range snap.Select(nano.MaxSpan) {
		select {
		case ch <- *seg:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (p *Pool) ScanPartitions(ctx context.Context, w zbuf.Writer, snap *commit.Snapshot, span nano.Span) error {
	ch := make(chan Partition, 10)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var err error
	go func() {
		err = ScanPartitions(ctx, snap, span, p.Order, ch)
		close(ch)
	}()
	m := zson.NewZNGMarshaler()
	m.Decorate(zson.StyleSimple)
	for p := range ch {
		rec, err := m.MarshalRecord(p)
		if err != nil {
			return err
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	return err
}

func (p *Pool) ScanSegments(ctx context.Context, w zbuf.Writer, snap *commit.Snapshot, span nano.Span) error {
	ch := make(chan segment.Reference)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var err error
	go func() {
		err = ScanSpan(ctx, snap, span, ch)
		close(ch)
	}()
	m := zson.NewZNGMarshaler()
	m.Decorate(zson.StyleSimple)
	for p := range ch {
		rec, err := m.MarshalRecord(p)
		if err != nil {
			return err
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	return err
}

func (p *Pool) IsJournalID(ctx context.Context, id journal.ID) (bool, error) {
	head, tail, err := p.log.Boundaries(ctx)
	if err != nil {
		return false, err
	}
	return id >= tail && id <= head, nil
}

func (p *Pool) Index(ctx context.Context, rules []index.Index, ids []ksuid.KSUID) (ksuid.KSUID, error) {
	idxrefs := make([]*index.Reference, 0, len(rules)*len(ids))
	for _, id := range ids {
		// This could be easily parallized with errgroup.
		refs, err := p.indexSegment(ctx, rules, id)
		if err != nil {
			return ksuid.Nil, err
		}
		idxrefs = append(idxrefs, refs...)
	}
	id := ksuid.New()
	txn := commit.NewAddIndicesTxn(id, idxrefs)
	if err := p.StoreInStaging(ctx, txn); err != nil {
		return ksuid.Nil, err
	}
	return id, nil
}

func (p *Pool) indexSegment(ctx context.Context, rules []index.Index, id ksuid.KSUID) ([]*index.Reference, error) {
	r, err := iosrc.NewReader(ctx, segment.RowObjectPath(p.DataPath, id))
	if err != nil {
		return nil, err
	}
	reader := zngio.NewReader(r, zson.NewContext())
	w, err := index.NewCombiner(ctx, p.IndexPath, rules, id)
	if err != nil {
		r.Close()
		return nil, err
	}
	err = zbuf.CopyWithContext(ctx, w, reader)
	if err != nil {
		w.Abort()
	} else {
		err = w.Close()
	}
	if rerr := r.Close(); err == nil {
		err = rerr
	}
	return w.References(), err
}

func DataPath(poolPath iosrc.URI) iosrc.URI {
	return poolPath.AppendPath(DataTag)
}

func StagePath(poolPath iosrc.URI) iosrc.URI {
	return poolPath.AppendPath(StageTag)
}

func LogPath(poolPath iosrc.URI) iosrc.URI {
	return poolPath.AppendPath(LogTag)
}

func IndexPath(poolPath iosrc.URI) iosrc.URI {
	return poolPath.AppendPath(IndexTag)
}

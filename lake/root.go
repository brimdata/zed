package lake

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zqe"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

// The Root of the lake represents the path prefix and configuration state
// for all of the data pools in the lake.  XXX For now, we are storing the
// pool configs in a json file without concurrency control.  We should
// make this use a journal and check for update conflicts.
type Root struct {
	path     iosrc.URI
	poolPath iosrc.URI
	// XXX Need local mutex on config
	Config
}

type Config struct {
	Version int           `zng:"version"`
	Pools   []PoolConfig  `zng:"pools"`
	Indices index.Indices `zng:"indices"`
}

func newRoot(path iosrc.URI) *Root {
	return &Root{
		path: path,
		//XXX For now this is just a json file with races,
		// but we'll eventually put this in a mutable journal.
		// See issue #2547.
		poolPath: path.AppendPath("pools.json"),
	}
}

func Open(ctx context.Context, path iosrc.URI) (*Root, error) {
	r := newRoot(path)
	if err := r.LoadConfig(ctx); err != nil {
		if zqe.IsNotFound(err) {
			err = fmt.Errorf("%s: no such lake", path)
		}
		return nil, err
	}
	return r, nil
}

func Create(ctx context.Context, path iosrc.URI) (*Root, error) {
	r := newRoot(path)
	if r.LoadConfig(ctx) == nil {
		return nil, fmt.Errorf("%s: lake already exists", path)
	}
	if err := iosrc.MkdirAll(path, 0700); err != nil {
		return nil, err
	}
	// Write an empty config file since StoreConfig uses iosrc.Replace()
	if err := iosrc.WriteFile(ctx, r.poolPath, nil); err != nil {
		return nil, err
	}
	if err := r.StoreConfig(ctx); err != nil {
		return nil, err
	}
	return r, nil
}

func CreateOrOpen(ctx context.Context, path iosrc.URI) (*Root, error) {
	r, err := Open(ctx, path)
	if err == nil {
		return r, err
	}
	return Create(ctx, path)
}

func (r *Root) LoadConfig(ctx context.Context) error {
	rc, err := iosrc.NewReader(ctx, r.poolPath)
	if err != nil {
		return err
	}
	defer rc.Close()
	if err := json.NewDecoder(rc).Decode(&r.Config); err != nil {
		return err
	}
	return nil
}

func (r *Root) StoreConfig(ctx context.Context) error {
	uri := r.poolPath
	err := iosrc.Replace(ctx, uri, func(w io.Writer) error {
		return json.NewEncoder(w).Encode(r.Config)
	})
	if err != nil {
		return err
	}
	if uri.Scheme == "file" {
		// Ensure the mtime is updated on the file after the close. This Chtimes
		// call was required due to failures seen in CI, when an mtime change
		// wasn't observed after some writes.
		// See https://github.com/brimdata/brim/issues/883.
		now := time.Now()
		return os.Chtimes(uri.Filepath(), now, now)
	}
	return nil
}

func (r *Root) ScanPools(ctx context.Context, w zio.Writer) error {
	m := zson.NewZNGMarshaler()
	m.Decorate(zson.StyleSimple)
	pools := r.Config.Pools
	for k := range r.Config.Pools {
		rec, err := m.MarshalRecord(&pools[k])
		if err != nil {
			return err
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	return nil
}

func (r *Root) LookupPool(_ context.Context, name string) *PoolConfig {
	for _, p := range r.Pools {
		if p.Name == name {
			return &p
		}
	}
	return nil
}

func (r *Root) OpenPool(ctx context.Context, name string) (*Pool, error) {
	poolRef := r.LookupPool(ctx, name)
	if poolRef == nil {
		return nil, fmt.Errorf("%s: pool not found", name)
	}
	return poolRef.Open(ctx, r.path)
}

func (r *Root) CreatePool(ctx context.Context, name string, keys []field.Static, order zbuf.Order, thresh int64) (*Pool, error) {
	// Pool creation can be a race so it's possible that two different
	// pools with the same name get created.  XXX mutex here won't protect
	// this because we can have distributed nodes created multiple pools
	// in the cloud object store.  That all said, this should be rare
	// and we can add logic to detect dupnames eventually and disable one
	// of them and warn the user.  You can always get at the underlying
	// pool using its ID.
	if r.LookupPool(ctx, name) != nil {
		return nil, fmt.Errorf("%s: pool already exists", name)
	}
	if thresh == 0 {
		thresh = segment.DefaultThreshold
	}
	poolRef := NewPoolConfig(name, ksuid.New(), keys, order, thresh)
	if err := poolRef.Create(ctx, r.path); err != nil {
		return nil, err
	}
	pool, err := poolRef.Open(ctx, r.path)
	if err != nil {
		return nil, err
	}
	r.Pools = append(r.Pools, *poolRef)
	if err := r.StoreConfig(ctx); err != nil {
		// XXX this is bad
		return nil, err
	}
	return pool, nil
}

// RemovePool removes all the each such directory and all of its contents.
func (r *Root) RemovePool(ctx context.Context, name string) error {
	hit := -1
	for k, p := range r.Pools {
		if p.Name == name {
			if hit >= 0 {
				return fmt.Errorf("multiple pools named %q: use pool ID to remove", name)
			}
			hit = k
		}
	}
	if hit < 0 {
		return fmt.Errorf("no such pool: %s", name)
	}
	if err := r.Pools[hit].Delete(ctx, r.path); err != nil {
		return err
	}
	r.Pools = append(r.Pools[:hit], r.Pools[hit+1:]...)
	return r.StoreConfig(ctx)
}

func (r *Root) AddIndex(ctx context.Context, indices []index.Index) error {
	updated := r.Indices
	for _, idx := range indices {
		var existing *index.Index
		if updated, existing = updated.Add(idx); existing != nil {
			return fmt.Errorf("index %s is a duplicate of index %s", idx.ID, existing.ID)
		}
	}
	r.Indices = updated
	return r.StoreConfig(ctx)
}

func (r *Root) DeleteIndices(ctx context.Context, ids []ksuid.KSUID) ([]index.Index, error) {
	updated := r.Indices
	deleted := make([]index.Index, len(ids))
	for i, id := range ids {
		var d *index.Index
		updated, d = updated.LookupDelete(id)
		if d == nil {
			return nil, fmt.Errorf("index %s not found", id)
		}
		deleted[i] = *d
	}
	return deleted, nil
}

func (r *Root) LookupIndices(ctx context.Context, ids []ksuid.KSUID) ([]index.Index, error) {
	indices := make([]index.Index, len(ids))
	for i, id := range ids {
		index := r.Indices.Lookup(id)
		if index == nil {
			return nil, fmt.Errorf("could not find index: %s", id)
		}
		indices[i] = *index
	}
	return indices, nil
}

func (r *Root) ListIndexIDs() []ksuid.KSUID {
	return r.Indices.IDs()
}

func (r *Root) ScanIndex(ctx context.Context, w zio.Writer, ids []ksuid.KSUID) error {
	m := zson.NewZNGMarshaler()
	m.Decorate(zson.StyleSimple)
	for _, id := range ids {
		index := r.Indices.Lookup(id)
		if index == nil {
			continue
		}
		rec, err := m.MarshalRecord(index)
		if err != nil {
			return err
		}
		if err := w.Write(rec); err != nil {
			return err
		}
	}
	return nil
}

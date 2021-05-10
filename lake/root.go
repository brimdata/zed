package lake

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/lake/commit"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zqe"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

var ErrPoolExists = errors.New("pool already exists")
var ErrPoolNotFound = errors.New("pool not found")

// The Root of the lake represents the path prefix and configuration state
// for all of the data pools in the lake.  XXX For now, we are storing the
// pool configs in a json file without concurrency control.  We should
// make this use a journal and check for update conflicts.
type Root struct {
	engine   storage.Engine
	path     *storage.URI
	poolPath *storage.URI
	configMu sync.Mutex
	Config
}

var _ proc.DataAdaptor = (*Root)(nil)

type Config struct {
	Version int           `zng:"version"`
	Pools   []PoolConfig  `zng:"pools"`
	Indices index.Indices `zng:"indices"`
}

func newRoot(engine storage.Engine, path *storage.URI) *Root {
	return &Root{
		engine: engine,
		path:   path,
		//XXX For now this is just a json file with races,
		// but we'll eventually put this in a mutable journal.
		// See issue #2547.
		poolPath: path.AppendPath("pools.json"),
	}
}

func Open(ctx context.Context, engine storage.Engine, path *storage.URI) (*Root, error) {
	r := newRoot(engine, path)
	if err := r.LoadConfig(ctx); err != nil {
		if zqe.IsNotFound(err) {
			err = fmt.Errorf("%s: no such lake", path)
		}
		return nil, err
	}
	return r, nil
}

func Create(ctx context.Context, engine storage.Engine, path *storage.URI) (*Root, error) {
	r := newRoot(engine, path)
	if r.LoadConfig(ctx) == nil {
		return nil, fmt.Errorf("%s: lake already exists", path)
	}
	//XXX For now, we write an empty config file to indicate that the
	// lake exists.  This will soon change to a lake config journal to
	// allow for write-concurrent atomic updates to config saved in
	// a shared cloud store.
	empty := storage.NewBytesReader(nil)
	if err := storage.Put(ctx, engine, r.poolPath, empty); err != nil {
		return nil, err
	}
	if err := r.StoreConfig(ctx); err != nil {
		return nil, err
	}
	return r, nil
}

func CreateOrOpen(ctx context.Context, engine storage.Engine, path *storage.URI) (*Root, error) {
	r, err := Open(ctx, engine, path)
	if err == nil {
		return r, err
	}
	return Create(ctx, engine, path)
}

func (r *Root) LoadConfig(ctx context.Context) error {
	r.configMu.Lock()
	defer r.configMu.Unlock()
	// XXX currently, we are storing the lake config using a file URL but
	// cloud instances will write the lake config to the cloud and generally
	// do not write to the file system (except for logs, /tmp, etc in which
	// case the storage engine isn't used).  For now, we just create a temp
	// local storage engine to deal with lake config.  We will fix this soon
	// when we switch the lake to config to use a key-value journal.
	local := storage.NewLocalEngine()
	b, err := storage.Get(ctx, local, r.poolPath)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &r.Config); err != nil {
		return err
	}
	return nil
}

func (r *Root) StoreConfig(ctx context.Context) error {
	r.configMu.Lock()
	defer r.configMu.Unlock()
	return r.storeConfig(ctx)
}

func (r *Root) storeConfig(ctx context.Context) error {
	uri := r.poolPath
	//XXX this will soon change to a key-value journal.  for now we write
	// the new config in its entirety and hope there is no error.
	b, err := json.Marshal(r.Config)
	if err != nil {
		return err
	}
	if err := storage.Put(ctx, r.engine, uri, bytes.NewReader(b)); err != nil {
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
	r.configMu.Lock()
	defer r.configMu.Unlock()
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

func (r *Root) ListPools() []PoolConfig {
	r.configMu.Lock()
	defer r.configMu.Unlock()
	return r.Config.Pools
}

func (r *Root) LookupPool(ctx context.Context, id ksuid.KSUID) *PoolConfig {
	r.configMu.Lock()
	defer r.configMu.Unlock()
	return r.lookupPool(ctx, id)
}

func (r *Root) lookupPool(_ context.Context, id ksuid.KSUID) *PoolConfig {
	for _, p := range r.Pools {
		if p.ID == id {
			return &p
		}
	}
	return nil
}

func (r *Root) LookupPoolByName(ctx context.Context, name string) *PoolConfig {
	r.configMu.Lock()
	defer r.configMu.Unlock()
	return r.lookupPoolByName(ctx, name)
}

func (r *Root) lookupPoolByName(_ context.Context, name string) *PoolConfig {
	for _, p := range r.Pools {
		if p.Name == name {
			return &p
		}
	}
	return nil
}

func (r *Root) Lookup(ctx context.Context, nameOrID string) (ksuid.KSUID, error) {
	if pool := r.LookupPoolByName(ctx, nameOrID); pool != nil {
		return pool.ID, nil
	}
	id, err := ksuid.Parse(nameOrID)
	if err != nil {
		return ksuid.Nil, err
	}
	return id, nil
}

func (r *Root) Layout(ctx context.Context, id ksuid.KSUID) (order.Layout, error) {
	p := r.LookupPool(ctx, id)
	if p == nil {
		return order.Nil, fmt.Errorf("no such pool ID: %s", id)
	}
	return p.Layout, nil
}

func (r *Root) OpenPool(ctx context.Context, id ksuid.KSUID) (*Pool, error) {
	poolRef := r.LookupPool(ctx, id)
	if poolRef == nil {
		return nil, ErrPoolNotFound
	}
	return poolRef.Open(ctx, r.engine, r.path)
}

func (r *Root) RenamePool(ctx context.Context, id ksuid.KSUID, newname string) error {
	r.configMu.Lock()
	defer r.configMu.Unlock()
	if exists := r.lookupPoolByName(ctx, newname); exists != nil {
		return ErrPoolExists
	}
	for i, p := range r.Pools {
		if p.ID == id {
			r.Pools[i].Name = newname
			return r.storeConfig(ctx)
		}
	}
	return fmt.Errorf("%s: %w", id, ErrPoolNotFound)
}

func (r *Root) CreatePool(ctx context.Context, name string, layout order.Layout, thresh int64) (*Pool, error) {
	r.configMu.Lock()
	defer r.configMu.Unlock()
	// Pool creation can be a race so it's possible that two different
	// pools with the same name get created.  XXX mutex here won't protect
	// this because we can have distributed nodes created multiple pools
	// in the cloud object store.  That all said, this should be rare
	// and we can add logic to detect dupnames eventually and disable one
	// of them and warn the user.  You can always get at the underlying
	// pool using its ID.
	if r.lookupPoolByName(ctx, name) != nil {
		return nil, fmt.Errorf("%s: %w", name, ErrPoolExists)
	}
	if thresh == 0 {
		thresh = segment.DefaultThreshold
	}
	poolRef := NewPoolConfig(name, ksuid.New(), layout, thresh)
	if err := poolRef.Create(ctx, r.engine, r.path); err != nil {
		return nil, err
	}
	pool, err := poolRef.Open(ctx, r.engine, r.path)
	if err != nil {
		return nil, err
	}
	r.Pools = append(r.Pools, *poolRef)
	if err := r.storeConfig(ctx); err != nil {
		// XXX this is bad
		return nil, err
	}
	return pool, nil
}

// RemovePool removes all the each such directory and all of its contents.
func (r *Root) RemovePool(ctx context.Context, id ksuid.KSUID) error {
	r.configMu.Lock()
	defer r.configMu.Unlock()
	hit := -1
	for k, p := range r.Pools {
		if p.ID == id {
			hit = k
			break
		}
	}
	if hit < 0 {
		return fmt.Errorf("%s: %w", id, ErrPoolNotFound)
	}
	if err := r.Pools[hit].Delete(ctx, r.engine, r.path); err != nil {
		return err
	}
	r.Pools = append(r.Pools[:hit], r.Pools[hit+1:]...)
	return r.storeConfig(ctx)
}

func (r *Root) AddIndex(ctx context.Context, indices []index.Index) error {
	r.configMu.Lock()
	defer r.configMu.Unlock()
	updated := r.Indices
	for _, idx := range indices {
		var existing *index.Index
		if updated, existing = updated.Add(idx); existing != nil {
			return fmt.Errorf("index %s is a duplicate of index %s", idx.ID, existing.ID)
		}
	}
	r.Indices = updated
	return r.storeConfig(ctx)
}

func (r *Root) DeleteIndices(ctx context.Context, ids []ksuid.KSUID) ([]index.Index, error) {
	r.configMu.Lock()
	defer r.configMu.Unlock()
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
	r.Indices = updated
	return deleted, r.storeConfig(ctx)
}

func (r *Root) LookupIndices(ctx context.Context, ids []ksuid.KSUID) ([]index.Index, error) {
	r.configMu.Lock()
	defer r.configMu.Unlock()
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
	r.configMu.Lock()
	defer r.configMu.Unlock()
	return r.Indices.IDs()
}

func (r *Root) ScanIndex(ctx context.Context, w zio.Writer, ids []ksuid.KSUID) error {
	r.configMu.Lock()
	defer r.configMu.Unlock()
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

func (r *Root) NewScheduler(ctx context.Context, zctx *zson.Context, p *dag.Pool, filter zbuf.Filter) (proc.Scheduler, error) {
	pool, err := r.OpenPool(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	var snap *commit.Snapshot
	if p.At != ksuid.Nil {
		id, err := pool.Log().JournalIDOfCommit(ctx, 0, p.At)
		if err != nil {
			return nil, err
		}
		snap, err = pool.log.Snapshot(ctx, id)
	} else {
		snap, err = pool.log.Head(ctx)
	}
	if err != nil {
		return nil, err
	}
	span := p.Span //XXX
	return NewSortedScheduler(ctx, zctx, pool, snap, span, filter), nil
}

func (r *Root) Open(context.Context, *zson.Context, string, zbuf.Filter) (zbuf.PullerCloser, error) {
	return nil, errors.New("cannot use 'file' source in a lake query")
}

func (r *Root) Get(context.Context, *zson.Context, string, zbuf.Filter) (zbuf.PullerCloser, error) {
	return nil, errors.New("'http' data source in a lake query is currently not supported")
}

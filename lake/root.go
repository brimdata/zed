package lake

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/lake/commit"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/journal/kvs"
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

const (
	PoolsTag      = "pools"
	IndexRulesTag = "index_rules"
)

// The Root of the lake represents the path prefix and configuration state
// for all of the data pools in the lake.
type Root struct {
	*Config
	engine storage.Engine
	path   *storage.URI
}

var _ proc.DataAdaptor = (*Root)(nil)

type Config struct {
	pools      *kvs.Store
	indexRules *index.Store
}

func newRoot(engine storage.Engine, path *storage.URI) *Root {
	return &Root{
		engine: engine,
		path:   path,
	}
}

func Open(ctx context.Context, engine storage.Engine, path *storage.URI) (*Root, error) {
	r := newRoot(engine, path)
	if err := r.loadConfig(ctx); err != nil {
		if zqe.IsNotFound(err) {
			err = fmt.Errorf("%s: no such lake", path)
		}
		return nil, err
	}
	return r, nil
}

func Create(ctx context.Context, engine storage.Engine, path *storage.URI) (*Root, error) {
	r := newRoot(engine, path)
	if err := r.loadConfig(ctx); err == nil {
		return nil, fmt.Errorf("%s: lake already exists", path)
	}
	c, err := r.createConfig(ctx)
	if err != nil {
		return nil, err
	}
	r.Config = c
	return r, nil
}

func CreateOrOpen(ctx context.Context, engine storage.Engine, path *storage.URI) (*Root, error) {
	r, err := Open(ctx, engine, path)
	if err == nil {
		return r, err
	}
	return Create(ctx, engine, path)
}

func (r *Root) createConfig(ctx context.Context) (*Config, error) {
	poolPath := r.path.AppendPath(PoolsTag)
	rulesPath := r.path.AppendPath(IndexRulesTag)
	types := []interface{}{PoolConfig{}}
	pools, err := kvs.Create(ctx, r.engine, poolPath, types)
	if err != nil {
		return nil, err
	}
	indexRules, err := index.CreateStore(ctx, r.engine, rulesPath)
	if err != nil {
		return nil, err
	}
	return &Config{pools, indexRules}, nil
}

func (r *Root) loadConfig(ctx context.Context) error {
	poolPath := r.path.AppendPath(PoolsTag)
	rulesPath := r.path.AppendPath(IndexRulesTag)
	types := []interface{}{PoolConfig{}}
	pools, err := kvs.Open(ctx, r.engine, poolPath, types)
	if err != nil {
		return err
	}
	indexRules, err := index.OpenStore(ctx, r.engine, rulesPath)
	if err != nil {
		return err
	}
	r.Config = &Config{pools, indexRules}
	return nil
}

func (r *Root) ScanPools(ctx context.Context, w zio.Writer) error {
	m := zson.NewZNGMarshaler()
	m.Decorate(zson.StyleSimple)
	pools, err := r.ListPools(ctx)
	if err != nil {
		return err
	}
	for k := range pools {
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

func (r *Root) ListPools(ctx context.Context) ([]PoolConfig, error) {
	entries, err := r.pools.All(ctx)
	if err != nil {
		return nil, err
	}
	pools := make([]PoolConfig, 0, len(entries))
	for _, entry := range entries {
		pool, ok := entry.Value.(*PoolConfig)
		if !ok {
			return nil, errors.New("corrupt pool config journal")
		}
		pools = append(pools, *pool)
	}
	return pools, nil
}

func lookupPool(pools []PoolConfig, fn func(PoolConfig) bool) *PoolConfig {
	for _, pool := range pools {
		if fn(pool) {
			return &pool
		}
	}
	return nil
}

func lookupPoolByName(pools []PoolConfig, name string) *PoolConfig {
	return lookupPool(pools, func(p PoolConfig) bool {
		return p.Name == name
	})
}

func (r *Root) LookupPool(ctx context.Context, id ksuid.KSUID) *PoolConfig {
	pools, err := r.ListPools(ctx)
	if err != nil {
		return nil
	}
	return lookupPool(pools, func(p PoolConfig) bool {
		return p.ID == id
	})
}

func (r *Root) LookupPoolByName(ctx context.Context, name string) *PoolConfig {
	pools, err := r.ListPools(ctx)
	if err != nil {
		return nil
	}
	return lookupPoolByName(pools, name)
}

func (r *Root) Lookup(ctx context.Context, nameOrID string) (ksuid.KSUID, error) {
	if pool := r.LookupPoolByName(ctx, nameOrID); pool != nil {
		return pool.ID, nil
	}
	id, err := ksuid.Parse(nameOrID)
	if err != nil {
		return ksuid.Nil, fmt.Errorf("%s: %w", nameOrID, ErrPoolNotFound)
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

func (r *Root) RenamePool(ctx context.Context, id ksuid.KSUID, newName string) error {
	pool := r.LookupPool(ctx, id)
	if pool == nil {
		return fmt.Errorf("%s: %w", id, ErrPoolNotFound)
	}
	oldName := pool.Name
	pool.Name = newName
	err := r.pools.Move(ctx, oldName, newName, pool)
	switch err {
	case kvs.ErrKeyExists:
		return ErrPoolExists
	case kvs.ErrNoSuchKey:
		return fmt.Errorf("%s: %w", id, ErrPoolNotFound)
	}
	return err
}

func (r *Root) CreatePool(ctx context.Context, name string, layout order.Layout, thresh int64) (*Pool, error) {
	if r.LookupPoolByName(ctx, name) != nil {
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
		poolRef.Delete(ctx, r.engine, r.path)
		return nil, err
	}
	if err := r.pools.Insert(ctx, name, poolRef); err != nil {
		poolRef.Delete(ctx, r.engine, r.path)
		if err == kvs.ErrKeyExists {
			return nil, fmt.Errorf("%s: %w", name, ErrPoolExists)
		}
		return nil, err
	}
	return pool, nil
}

// RemovePool deletes a pool from the configuration journal and deletes all
// data associated with the pool.
func (r *Root) RemovePool(ctx context.Context, id ksuid.KSUID) error {
	poolConfig := r.LookupPool(ctx, id)
	if poolConfig == nil {
		return fmt.Errorf("%s: %w", id, ErrPoolNotFound)
	}
	err := r.pools.Delete(ctx, poolConfig.Name, func(v interface{}) bool {
		p, ok := v.(*PoolConfig)
		if !ok {
			return false
		}
		return p.ID == id
	})
	if err != nil {
		if err == kvs.ErrNoSuchKey {
			return fmt.Errorf("%s: %w", id, ErrPoolNotFound)
		}
		if err == kvs.ErrConstraint {
			return fmt.Errorf("%s: pool %q renamed during removal", poolConfig.Name, id)
		}
		return err
	}
	return poolConfig.Delete(ctx, r.engine, r.path)
}

func (r *Root) AddIndexRules(ctx context.Context, rules []index.Rule) error {
	//XXX should change this to do a single commit for all of the rules
	// and abort all if one fails.  (change Add() semantics)
	for _, rule := range rules {
		if err := r.indexRules.Add(ctx, rule); err != nil {
			return err
		}
	}
	return nil
}

func (r *Root) DeleteIndexRules(ctx context.Context, ids []ksuid.KSUID) ([]index.Rule, error) {
	deleted := make([]index.Rule, 0, len(ids))
	for _, id := range ids {
		rule, err := r.indexRules.Delete(ctx, id)
		if err != nil {
			return deleted, fmt.Errorf("index %s not found", id)
		}
		deleted = append(deleted, rule)
	}
	return deleted, nil
}

func (r *Root) LookupIndexRules(ctx context.Context, name string) ([]index.Rule, error) {
	return r.indexRules.Lookup(ctx, name)
}

func (r *Root) ScanIndexRules(ctx context.Context, w zio.Writer, names []string) error {
	m := zson.NewZNGMarshaler()
	m.Decorate(zson.StyleSimple)
	if len(names) == 0 {
		var err error
		names, err = r.indexRules.Names(ctx)
		if err != nil {
			return err
		}
	}
	for _, name := range names {
		rules, err := r.indexRules.Lookup(ctx, name)
		if err != nil {
			if err == index.ErrNoSuchRule {
				continue
			}
			return err
		}
		sort.Slice(rules, func(i, j int) bool {
			return rules[i].CreateTime() < rules[j].CreateTime()
		})
		for _, rule := range rules {
			rec, err := m.MarshalRecord(rule)
			if err != nil {
				return err
			}
			if err := w.Write(rec); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Root) NewScheduler(ctx context.Context, zctx *zson.Context, id, at ksuid.KSUID, span extent.Span, filter zbuf.Filter) (proc.Scheduler, error) {
	pool, err := r.OpenPool(ctx, id)
	if err != nil {
		return nil, err
	}
	var snap *commit.Snapshot
	if at != ksuid.Nil {
		id, err := pool.Log().JournalIDOfCommit(ctx, 0, at)
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
	return NewSortedScheduler(ctx, zctx, pool, snap, span, filter), nil
}

func (r *Root) Open(context.Context, *zson.Context, string, zbuf.Filter) (zbuf.PullerCloser, error) {
	return nil, errors.New("cannot use 'file' or 'http' source in a lake query")
}

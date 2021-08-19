package lake

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/lake/commit"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/lake/journal/kvs"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zngbytes"
	"github.com/brimdata/zed/zqe"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

var (
	ErrPoolExists   = errors.New("pool already exists")
	ErrPoolNotFound = errors.New("pool not found")
)

const (
	Version         = 1
	PoolsTag        = "pools"
	MetaTag         = "metas"
	IndexRulesTag   = "index_rules"
	LakeMagicFile   = "lake.zng"
	LakeMagicString = "ZED LAKE"
)

// The Root of the lake represents the path prefix and configuration state
// for all of the data pools in the lake.
type Root struct {
	engine     storage.Engine
	path       *storage.URI
	pools      *kvs.Store
	indexRules *index.Store
}

var _ proc.DataAdaptor = (*Root)(nil)

type LakeMagic struct {
	Magic   string `zng:"magic"`
	Version int    `zng:"version"`
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
	if err := r.createConfig(ctx); err != nil {
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

func (r *Root) createConfig(ctx context.Context) error {
	poolPath := r.path.AppendPath(PoolsTag)
	rulesPath := r.path.AppendPath(IndexRulesTag)
	types := []interface{}{PoolConfig{}}
	var err error
	r.pools, err = kvs.Create(ctx, r.engine, poolPath, types)
	if err != nil {
		return err
	}
	r.indexRules, err = index.CreateStore(ctx, r.engine, rulesPath)
	if err != nil {
		return err
	}
	if err := r.writeLakeMagic(ctx); err != nil {
		return err
	}
	return err
}

func (r *Root) loadConfig(ctx context.Context) error {
	if err := r.readLakeMagic(ctx); err != nil {
		return err
	}
	poolPath := r.path.AppendPath(PoolsTag)
	rulesPath := r.path.AppendPath(IndexRulesTag)
	types := []interface{}{PoolConfig{}}
	var err error
	r.pools, err = kvs.Open(ctx, r.engine, poolPath, types)
	if err != nil {
		return err
	}
	r.indexRules, err = index.OpenStore(ctx, r.engine, rulesPath)
	return err
}

func (r *Root) writeLakeMagic(ctx context.Context) error {
	if err := r.readLakeMagic(ctx); err == nil {
		return errors.New("lake already exists")
	}
	magic := &LakeMagic{
		Magic:   LakeMagicString,
		Version: Version,
	}
	serializer := zngbytes.NewSerializer()
	if err := serializer.Write(magic); err != nil {
		return err
	}
	if err := serializer.Close(); err != nil {
		return err
	}
	path := r.path.AppendPath(LakeMagicFile)
	err := r.engine.PutIfNotExists(ctx, path, serializer.Bytes())
	if err == storage.ErrNotSupported {
		//XXX workaround for now: see issue #2686
		reader := bytes.NewReader(serializer.Bytes())
		err = storage.Put(ctx, r.engine, path, reader)
	}
	return err
}

func (r *Root) readLakeMagic(ctx context.Context) error {
	path := r.path.AppendPath(LakeMagicFile)
	reader, err := r.engine.Get(ctx, path)
	if err != nil {
		return err
	}
	deserializer := zngbytes.NewDeserializer(reader, []interface{}{
		LakeMagic{},
	})
	v, err := deserializer.Read()
	if err != nil {
		return err
	}
	magic, ok := v.(*LakeMagic)
	if !ok {
		return fmt.Errorf("corrupt lake magic file %q: unknown type: %T", LakeMagicFile, v)
	}
	if magic.Magic != LakeMagicString {
		return fmt.Errorf("corrupt lake magic: %q should be %q", magic.Magic, LakeMagicString)
	}
	if magic.Version != Version {
		return fmt.Errorf("unsupported lake version: found version %d while expecting %d", magic.Version, Version)
	}
	return nil
}

func (r *Root) batchifyPools(ctx context.Context, zctx *zson.Context, f expr.Filter) (zbuf.Array, error) {
	m := zson.NewZNGMarshalerWithContext(zctx)
	m.Decorate(zson.StyleSimple)
	pools, err := r.ListPools(ctx)
	if err != nil {
		return nil, err
	}
	var batch zbuf.Array
	for k := range pools {
		rec, err := m.MarshalRecord(&pools[k])
		if err != nil {
			return nil, err
		}
		if f == nil || f(rec) {
			batch.Append(rec)
		}
	}
	return batch, nil
}

//XXX ScanPools will go away with issue #2953
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

func (r *Root) LookupIDs(ctx context.Context, poolName string, branchName string) (ksuid.KSUID, ksuid.KSUID, error) {
	if poolName == "" {
		return ksuid.Nil, ksuid.Nil, errors.New("no pool name given")
	}
	poolID, err := ksuid.Parse(poolName)
	if err != nil {
		pool := r.LookupPoolByName(ctx, poolName)
		if pool == nil {
			return ksuid.Nil, ksuid.Nil, fmt.Errorf("%s: pool not found", poolName)
		}
		poolID = pool.ID
	}
	// XXX need to look up branch ID
	return poolID, ksuid.Nil, nil
}

func (r *Root) Layout(ctx context.Context, src dag.Source) order.Layout {
	poolSrc, ok := src.(*dag.Pool)
	if !ok {
		return order.Nil
	}
	pool := r.LookupPool(ctx, poolSrc.ID)
	if pool == nil {
		return order.Nil
	}
	return pool.Layout
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

func (r *Root) batchifyIndexRules(ctx context.Context, zctx *zson.Context, f expr.Filter) (zbuf.Array, error) {
	m := zson.NewZNGMarshalerWithContext(zctx)
	m.Decorate(zson.StyleSimple)
	names, err := r.indexRules.Names(ctx)
	if err != nil {
		return nil, err
	}
	var batch zbuf.Array
	for _, name := range names {
		rules, err := r.indexRules.Lookup(ctx, name)
		if err != nil {
			if err == index.ErrNoSuchRule {
				continue
			}
			return nil, err
		}
		sort.Slice(rules, func(i, j int) bool {
			return rules[i].CreateTime() < rules[j].CreateTime()
		})
		for _, rule := range rules {
			rec, err := m.MarshalRecord(rule)
			if err != nil {
				return nil, err
			}
			if f == nil || !f(rec) {
				batch.Append(rec)
			}
		}
	}
	return batch, nil
}

func (r *Root) NewScheduler(ctx context.Context, zctx *zson.Context, src dag.Source, span extent.Span, filter zbuf.Filter) (proc.Scheduler, error) {
	switch src := src.(type) {
	case *dag.Pool:
		return r.newPoolScheduler(ctx, zctx, src.ID, src.At, span, filter)
	case *dag.LakeMeta:
		return r.newLakeMetaScheduler(ctx, zctx, src.Meta, filter)
	case *dag.PoolMeta:
		return r.newPoolMetaScheduler(ctx, zctx, src.ID, src.Meta, src.At, span, filter)
	default:
		return nil, fmt.Errorf("internal error: unsupported source type in lake.Root.NewScheduler(): %T", src)
	}
}

func (r *Root) newLakeMetaScheduler(ctx context.Context, zctx *zson.Context, meta string, filter zbuf.Filter) (proc.Scheduler, error) {
	f, err := filter.AsFilter()
	if err != nil {
		return nil, err
	}
	var batch zbuf.Array
	switch meta {
	case "pools":
		batch, err = r.batchifyPools(ctx, zctx, f)
	case "index_rules":
		batch, err = r.batchifyIndexRules(ctx, zctx, f)
	default:
		return nil, fmt.Errorf("unknown lake metadata type: %q", meta)
	}
	if err != nil {
		return nil, err
	}
	s, err := zbuf.NewScanner(ctx, &batch, filter)
	if err != nil {
		return nil, err
	}
	return newScannerScheduler(s), nil
}

func (r *Root) newPoolMetaScheduler(ctx context.Context, zctx *zson.Context, poolID ksuid.KSUID, meta string, tag ksuid.KSUID, span extent.Span, filter zbuf.Filter) (proc.Scheduler, error) {
	p, err := r.OpenPool(ctx, poolID)
	if err != nil {
		return nil, err
	}
	switch meta {
	case "objects":
		snap, err := p.SnapshotOf(ctx, tag)
		if err != nil {
			return nil, err
		}
		reader, err := p.readerOfObjects(ctx, zctx, snap, span)
		if err != nil {
			return nil, err
		}
		s, err := zbuf.NewScanner(ctx, reader, filter)
		if err != nil {
			return nil, err
		}
		return newScannerScheduler(s), nil
	case "partitions":
		snap, err := p.SnapshotOf(ctx, tag)
		if err != nil {
			return nil, err
		}
		reader, err := p.readerOfPartitions(ctx, zctx, snap, span)
		if err != nil {
			return nil, err
		}
		s, err := zbuf.NewScanner(ctx, reader, filter)
		if err != nil {
			return nil, err
		}
		return newScannerScheduler(s), nil
	case "log":
		reader, err := p.Log().OpenAsZNG(ctx, zctx, journal.Nil, journal.Nil)
		if err != nil {
			return nil, err
		}
		s, err := zbuf.NewScanner(ctx, reader, filter)
		if err != nil {
			return nil, err
		}
		return newScannerScheduler(s), nil
	default:
		return nil, fmt.Errorf("unknown pool metadata type: %q", meta)
	}
}

func (r *Root) newPoolScheduler(ctx context.Context, zctx *zson.Context, id, at ksuid.KSUID, span extent.Span, filter zbuf.Filter) (proc.Scheduler, error) {
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

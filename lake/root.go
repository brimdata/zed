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
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/lake/journal/kvs"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zngbytes"
	"github.com/brimdata/zed/zqe"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

var (
	ErrPoolExists     = errors.New("pool already exists")
	ErrPoolNotFound   = errors.New("pool not found")
	ErrBranchExists   = errors.New("branch already exists")
	ErrBranchNotFound = errors.New("branch not found")
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
	return r.writeLakeMagic(ctx)
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
		return fmt.Errorf("corrupt lake version file %q: unknown type: %T", LakeMagicFile, v)
	}
	if magic.Magic != LakeMagicString {
		return fmt.Errorf("corrupt lake version file: magic %q should be %q", magic.Magic, LakeMagicString)
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

func (r *Root) batchifyBranches(ctx context.Context, zctx *zson.Context, f expr.Filter) (zbuf.Array, error) {
	m := zson.NewZNGMarshalerWithContext(zctx)
	m.Decorate(zson.StyleSimple)
	poolRefs, err := r.ListPools(ctx)
	if err != nil {
		return nil, err
	}
	var batch zbuf.Array
	for _, poolRef := range poolRefs {
		pool, err := poolRef.Open(ctx, r.engine, r.path)
		if err != nil {
			// We could have race here because a pool got deleted
			// while we looped so we check and continue.
			if errors.Is(err, ErrPoolNotFound) {
				continue
			}
			return nil, err
		}
		batch, err = pool.batchifyBranches(ctx, batch, m, f)
		if err != nil {
			return nil, err
		}
	}
	return batch, nil
}

type BranchMeta struct {
	PoolConfig   `zng:"pool"`
	BranchConfig `zng:"branch"`
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

func (r *Root) IDs(ctx context.Context, poolName string, branchName string) (ksuid.KSUID, ksuid.KSUID, error) {
	if poolName == "" {
		return ksuid.Nil, ksuid.Nil, errors.New("no pool name given")
	}
	poolID, err := ksuid.Parse(poolName)
	var poolRef *PoolConfig
	if err != nil {
		poolRef = r.LookupPoolByName(ctx, poolName)
		if poolRef == nil {
			return ksuid.Nil, ksuid.Nil, fmt.Errorf("%s: pool not found", poolName)
		}
		poolID = poolRef.ID
	}
	if branchName == "" {
		return poolID, ksuid.Nil, nil
	}
	branchID, err := ksuid.Parse(branchName)
	if err != nil {
		if poolRef == nil {
			poolRef = r.LookupPool(ctx, poolID)
		}
		pool, err := poolRef.Open(ctx, r.engine, r.path)
		if err != nil {
			return ksuid.Nil, ksuid.Nil, err
		}
		branchRef, err := pool.LookupBranchByName(ctx, branchName)
		if err != nil {
			return ksuid.Nil, ksuid.Nil, err
		}
		branchID = branchRef.ID
	}
	return poolID, branchID, nil
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
	poolRef := NewPoolConfig(name, layout, thresh)
	if err := poolRef.Create(ctx, r.engine, r.path); err != nil {
		return nil, err
	}
	pool, err := poolRef.Open(ctx, r.engine, r.path)
	if err != nil {
		poolRef.Remove(ctx, r.engine, r.path)
		return nil, err
	}
	if err := r.pools.Insert(ctx, name, poolRef); err != nil {
		poolRef.Remove(ctx, r.engine, r.path)
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
	return poolConfig.Remove(ctx, r.engine, r.path)
}

func (r *Root) CreateBranch(ctx context.Context, poolID ksuid.KSUID, name string, parent, atTag ksuid.KSUID) (*BranchConfig, error) {
	poolRef := r.LookupPool(ctx, poolID)
	if poolRef == nil {
		return nil, fmt.Errorf("%s: %w", poolID, ErrPoolNotFound)
	}
	pool, err := poolRef.Open(ctx, r.engine, r.path)
	if err != nil {
		return nil, err
	}
	var at journal.ID
	if parent != ksuid.Nil {
		parentBranch, err := pool.OpenBranchByID(ctx, parent)
		if err != nil {
			return nil, err
		}
		if atTag != ksuid.Nil {
			at, err = parentBranch.log.JournalIDOfCommit(ctx, 0, atTag)
			if err != nil {
				return nil, fmt.Errorf("%s: no such commit ID in %s/%s", atTag, poolRef.Name, parentBranch.Name)
			}
		} else {
			at, err = parentBranch.log.TipOfJournal(ctx)
			if err != nil {
				return nil, err
			}
		}
	}
	return poolRef.createBranch(ctx, r.engine, r.path, name, parent, at)
}

func (r *Root) RemoveBranch(ctx context.Context, poolID, branchID ksuid.KSUID) error {
	pool, err := r.OpenPool(ctx, poolID)
	if err != nil {
		return err
	}
	return pool.removeBranch(ctx, branchID)
}

// MergeBranch merges the indicated branch into its parent returning the
// commit tag of the new commit into the parent branch.
func (r *Root) MergeBranch(ctx context.Context, poolID, branchID, at ksuid.KSUID) (ksuid.KSUID, error) {
	pool, err := r.OpenPool(ctx, poolID)
	if err != nil {
		return ksuid.Nil, err
	}
	branch, err := pool.OpenBranchByID(ctx, branchID)
	if err != nil {
		return ksuid.Nil, err
	}
	return branch.Merge(ctx, at)
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
		return r.newPoolScheduler(ctx, zctx, src.ID, src.Branch, src.At, span, filter)
	case *dag.LakeMeta:
		return r.newLakeMetaScheduler(ctx, zctx, src.Meta, filter)
	case *dag.PoolMeta:
		return r.newPoolMetaScheduler(ctx, zctx, src.ID, src.Meta, filter)
	case *dag.BranchMeta:
		return r.newBranchMetaScheduler(ctx, zctx, src.ID, src.Branch, src.Meta, src.At, span, filter)
	default:
		return nil, fmt.Errorf("internal error: unsupported source type in lake.Root.NewScheduler: %T", src)
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
	case "branches":
		batch, err = r.batchifyBranches(ctx, zctx, f)
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

func (r *Root) newPoolMetaScheduler(ctx context.Context, zctx *zson.Context, poolID ksuid.KSUID, meta string, filter zbuf.Filter) (proc.Scheduler, error) {
	f, err := filter.AsFilter()
	if err != nil {
		return nil, err
	}
	p, err := r.OpenPool(ctx, poolID)
	if err != nil {
		return nil, err
	}
	var batch zbuf.Array
	switch meta {
	case "branches":
		m := zson.NewZNGMarshalerWithContext(zctx)
		m.Decorate(zson.StyleSimple)
		batch, err = p.batchifyBranches(ctx, batch, m, f)
	default:
		return nil, fmt.Errorf("unknown pool metadata type: %q", meta)
	}
	s, err := zbuf.NewScanner(ctx, &batch, filter)
	if err != nil {
		return nil, err
	}
	return newScannerScheduler(s), nil
}

func (r *Root) newBranchMetaScheduler(ctx context.Context, zctx *zson.Context, poolID, branchID ksuid.KSUID, meta string, tag ksuid.KSUID, span extent.Span, filter zbuf.Filter) (proc.Scheduler, error) {
	p, err := r.OpenPool(ctx, poolID)
	if err != nil {
		return nil, err
	}
	branch, err := p.OpenBranchByID(ctx, branchID)
	if err != nil {
		return nil, err
	}
	switch meta {
	case "objects":
		snap, err := branch.Snapshot(ctx, tag)
		if err != nil {
			return nil, err
		}
		reader, err := objectReader(ctx, zctx, snap, span, p.Layout.Order)
		if err != nil {
			return nil, err
		}
		s, err := zbuf.NewScanner(ctx, reader, filter)
		if err != nil {
			return nil, err
		}
		return newScannerScheduler(s), nil
	case "partitions":
		snap, err := branch.Snapshot(ctx, tag)
		if err != nil {
			return nil, err
		}
		reader, err := partitionReader(ctx, zctx, snap, span, p.Layout.Order)
		if err != nil {
			return nil, err
		}
		s, err := zbuf.NewScanner(ctx, reader, filter)
		if err != nil {
			return nil, err
		}
		return newScannerScheduler(s), nil
	case "log":
		reader, err := branch.Log().OpenAsZNG(ctx, zctx, journal.Nil, journal.Nil)
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

func (r *Root) newPoolScheduler(ctx context.Context, zctx *zson.Context, poolID, branchID, at ksuid.KSUID, span extent.Span, filter zbuf.Filter) (proc.Scheduler, error) {
	pool, err := r.OpenPool(ctx, poolID)
	if err != nil {
		return nil, err
	}
	return pool.newScheduler(ctx, zctx, branchID, at, span, filter)
}

func (r *Root) Open(context.Context, *zson.Context, string, zbuf.Filter) (zbuf.PullerCloser, error) {
	return nil, errors.New("cannot use 'file' or 'http' source in a lake query")
}

package lake

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sort"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/lake/branches"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/pools"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zngbytes"
	"github.com/brimdata/zed/zson"
	lru "github.com/hashicorp/golang-lru"
	"github.com/segmentio/ksuid"
)

const (
	Version         = 1
	PoolsTag        = "pools"
	IndexRulesTag   = "index_rules"
	LakeMagicFile   = "lake.zng"
	LakeMagicString = "ZED LAKE"
)

// The Root of the lake represents the path prefix and configuration state
// for all of the data pools in the lake.
type Root struct {
	engine     storage.Engine
	path       *storage.URI
	poolCache  *lru.ARCCache // Used like a map[ksuid.KSUID]*Pool.
	pools      *pools.Store
	indexRules *index.Store
}

type LakeMagic struct {
	Magic   string `zed:"magic"`
	Version int    `zed:"version"`
}

func newRoot(engine storage.Engine, path *storage.URI) *Root {
	poolCache, err := lru.NewARC(1024)
	if err != nil {
		panic(err)
	}
	return &Root{
		engine:    engine,
		path:      path,
		poolCache: poolCache,
	}
}

func Open(ctx context.Context, engine storage.Engine, path *storage.URI) (*Root, error) {
	r := newRoot(engine, path)
	if err := r.loadConfig(ctx); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
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
	var err error
	r.pools, err = pools.CreateStore(ctx, r.engine, poolPath)
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
	var err error
	r.pools, err = pools.OpenStore(ctx, r.engine, poolPath)
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
	serializer.Decorate(zson.StylePackage)
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
	defer deserializer.Close()
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

func (r *Root) BatchifyPools(ctx context.Context, zctx *zed.Context, f expr.Evaluator) ([]zed.Value, error) {
	m := zson.NewZNGMarshalerWithContext(zctx)
	m.Decorate(zson.StylePackage)
	pools, err := r.ListPools(ctx)
	if err != nil {
		return nil, err
	}
	ectx := expr.NewContext()
	var vals []zed.Value
	for k := range pools {
		rec, err := m.MarshalRecord(&pools[k])
		if err != nil {
			return nil, err
		}
		if filter(zctx, ectx, rec, f) {
			vals = append(vals, *rec)
		}
	}
	return vals, nil
}

func (r *Root) BatchifyBranches(ctx context.Context, zctx *zed.Context, f expr.Evaluator) ([]zed.Value, error) {
	m := zson.NewZNGMarshalerWithContext(zctx)
	m.Decorate(zson.StylePackage)
	poolRefs, err := r.ListPools(ctx)
	if err != nil {
		return nil, err
	}
	var vals []zed.Value
	for k := range poolRefs {
		pool, err := r.openPool(ctx, &poolRefs[k])
		if err != nil {
			// We could have race here because a pool got deleted
			// while we looped so we check and continue.
			if errors.Is(err, pools.ErrNotFound) {
				continue
			}
			return nil, err
		}
		vals, err = pool.BatchifyBranches(ctx, zctx, vals, m, f)
		if err != nil {
			return nil, err
		}
	}
	return vals, nil
}

type BranchMeta struct {
	Pool   pools.Config    `zed:"pool"`
	Branch branches.Config `zed:"branch"`
}

func (r *Root) ListPools(ctx context.Context) ([]pools.Config, error) {
	return r.pools.All(ctx)
}

func (r *Root) PoolID(ctx context.Context, poolName string) (ksuid.KSUID, error) {
	if poolName == "" {
		return ksuid.Nil, errors.New("no pool name given")
	}
	poolID, err := ksuid.Parse(poolName)
	var poolRef *pools.Config
	if err != nil {
		poolRef = r.pools.LookupByName(ctx, poolName)
		if poolRef == nil {
			return ksuid.Nil, fmt.Errorf("%s: %w", poolName, pools.ErrNotFound)
		}
		poolID = poolRef.ID
	}
	return poolID, nil
}

func (r *Root) CommitObject(ctx context.Context, poolID ksuid.KSUID, branchName string) (ksuid.KSUID, error) {
	pool, err := r.OpenPool(ctx, poolID)
	if err != nil {
		return ksuid.Nil, err
	}
	branchRef, err := pool.LookupBranchByName(ctx, branchName)
	if err != nil {
		return ksuid.Nil, err
	}
	return branchRef.Commit, nil
}

func (r *Root) Layout(ctx context.Context, src dag.Source) order.Layout {
	/*poolSrc, ok := src.(*dag.Pool) XXX
	if !ok {
		return order.Nil
	}
	*/
	//config, err := r.pools.LookupByID(ctx, poolSrc.ID)
	config, err := r.pools.LookupByID(ctx, ksuid.Nil) //XXX temp to compile
	if err != nil {
		return order.Nil
	}
	return config.Layout
}

func (r *Root) OpenPool(ctx context.Context, id ksuid.KSUID) (*Pool, error) {
	config, err := r.pools.LookupByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.openPool(ctx, config)
}

func (r *Root) openPool(ctx context.Context, config *pools.Config) (*Pool, error) {
	if v, ok := r.poolCache.Get(config.ID); ok {
		// The cached pool's config may be outdated, so rather than
		// return the pool directly, we return a copy whose config we
		// can safely update without locking.
		p := *v.(*Pool)
		p.Config = *config
		return &p, nil
	}
	p, err := OpenPool(ctx, config, r.engine, r.path)
	if err != nil {
		return nil, err
	}
	r.poolCache.Add(config.ID, p)
	return p, nil
}

func (r *Root) RenamePool(ctx context.Context, id ksuid.KSUID, newName string) error {
	return r.pools.Rename(ctx, id, newName)
}

func (r *Root) CreatePool(ctx context.Context, name string, layout order.Layout, seekStride int, thresh int64) (*Pool, error) {
	if name == "HEAD" {
		return nil, fmt.Errorf("pool cannot be named %q", name)
	}
	if r.pools.LookupByName(ctx, name) != nil {
		return nil, fmt.Errorf("%s: %w", name, pools.ErrExists)
	}
	if thresh == 0 {
		thresh = data.DefaultThreshold
	}
	config := pools.NewConfig(name, layout, thresh, seekStride)
	if err := CreatePool(ctx, config, r.engine, r.path); err != nil {
		return nil, err
	}
	pool, err := r.openPool(ctx, config)
	if err != nil {
		RemovePool(ctx, config, r.engine, r.path)
		return nil, err
	}
	if err := r.pools.Add(ctx, config); err != nil {
		RemovePool(ctx, config, r.engine, r.path)
		return nil, err
	}
	return pool, nil
}

// RemovePool deletes a pool from the configuration journal and deletes all
// data associated with the pool.
func (r *Root) RemovePool(ctx context.Context, id ksuid.KSUID) error {
	config, err := r.pools.LookupByID(ctx, id)
	if err != nil {
		return err
	}
	if err := r.pools.Remove(ctx, *config); err != nil {
		return err
	}
	// This pool might be cached on other cluster nodes, but that's fine.
	// With no entry in the pool store, it will be inaccessible and
	// eventually evicted by the cache's LRU algorithm.
	r.poolCache.Remove(config.ID)
	return RemovePool(ctx, config, r.engine, r.path)
}

func (r *Root) CreateBranch(ctx context.Context, poolID ksuid.KSUID, name string, parent ksuid.KSUID) (*branches.Config, error) {
	config, err := r.pools.LookupByID(ctx, poolID)
	if err != nil {
		return nil, err
	}
	return CreateBranch(ctx, config, r.engine, r.path, name, parent)
}

func (r *Root) RemoveBranch(ctx context.Context, poolID ksuid.KSUID, name string) error {
	pool, err := r.OpenPool(ctx, poolID)
	if err != nil {
		return err
	}
	return pool.removeBranch(ctx, name)
}

// MergeBranch merges the indicated branch into its parent returning the
// commit tag of the new commit into the parent branch.
func (r *Root) MergeBranch(ctx context.Context, poolID ksuid.KSUID, childBranch, parentBranch, author, message string) (ksuid.KSUID, error) {
	pool, err := r.OpenPool(ctx, poolID)
	if err != nil {
		return ksuid.Nil, err
	}
	child, err := pool.OpenBranchByName(ctx, childBranch)
	if err != nil {
		return ksuid.Nil, err
	}
	parent, err := pool.OpenBranchByName(ctx, parentBranch)
	if err != nil {
		return ksuid.Nil, err
	}
	return child.mergeInto(ctx, parent, author, message)
}

func (r *Root) Revert(ctx context.Context, poolID ksuid.KSUID, branchName string, commitID ksuid.KSUID, author, message string) (ksuid.KSUID, error) {
	pool, err := r.OpenPool(ctx, poolID)
	if err != nil {
		return ksuid.Nil, err
	}
	branch, err := pool.OpenBranchByName(ctx, branchName)
	if err != nil {
		return ksuid.Nil, err
	}
	return branch.Revert(ctx, commitID, author, message)
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

func (r *Root) LookupIndexRules(ctx context.Context, names ...string) ([]index.Rule, error) {
	var rules []index.Rule
	for _, name := range names {
		r, err := r.indexRules.Lookup(ctx, name)
		if err != nil {
			return nil, err
		}
		rules = append(rules, r...)
	}
	return rules, nil
}

func (r *Root) AllIndexRules(ctx context.Context) ([]index.Rule, error) {
	return r.indexRules.All(ctx)
}

func (r *Root) BatchifyIndexRules(ctx context.Context, zctx *zed.Context, f expr.Evaluator) ([]zed.Value, error) {
	m := zson.NewZNGMarshalerWithContext(zctx)
	m.Decorate(zson.StylePackage)
	names, err := r.indexRules.Names(ctx)
	if err != nil {
		return nil, err
	}
	var vals []zed.Value
	ectx := expr.NewContext()
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
			if filter(zctx, ectx, rec, f) {
				vals = append(vals, *rec)
			}
		}
	}
	return vals, nil
}

func (r *Root) Open(context.Context, *zed.Context, string, string, zbuf.Filter) (zbuf.Puller, error) {
	return nil, errors.New("cannot use 'file' or 'http' source in a lake query")
}

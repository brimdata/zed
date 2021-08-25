package lake

import (
	"context"
	"fmt"

	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/lake/commit"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/lake/journal/kvs"
	"github.com/brimdata/zed/lake/segment"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

const (
	DataTag     = "data"
	IndexTag    = "index"
	BranchesTag = "branches"
)

type PoolConfig struct {
	Name      string       `zng:"name"`
	ID        ksuid.KSUID  `zng:"id"`
	Layout    order.Layout `zng:"layout"`
	Threshold int64        `zng:"threshold"`
}

type Pool struct {
	PoolConfig
	engine       storage.Engine
	Path         *storage.URI
	DataPath     *storage.URI
	IndexPath    *storage.URI
	BranchesPath *storage.URI
	branches     *kvs.Store
}

func NewPoolConfig(name string, layout order.Layout, thresh int64) *PoolConfig {
	if thresh == 0 {
		thresh = segment.DefaultThreshold
	}
	return &PoolConfig{
		Name:      name,
		ID:        ksuid.New(),
		Layout:    layout,
		Threshold: thresh,
	}
}

func (p *PoolConfig) Path(root *storage.URI) *storage.URI {
	return root.AppendPath(p.ID.String())
}

var branchStoreTypes = []interface{}{BranchConfig{}}

func (p *PoolConfig) Create(ctx context.Context, engine storage.Engine, root *storage.URI) error {
	poolPath := p.Path(root)
	// branchesPath is the path to the kvs journal of BranchConfigs
	// for the pool while the commit log is stored in <pool-id>/<branch-id>.
	branchesPath := poolPath.AppendPath(BranchesTag)
	// create the branches journal store
	_, err := kvs.Create(ctx, engine, branchesPath, branchStoreTypes)
	if err != nil {
		return err
	}
	// create the main branche in the branches journal store
	_, err = p.createBranch(ctx, engine, root, "main", ksuid.Nil, journal.Nil)
	return err
}

func (p *PoolConfig) createBranch(ctx context.Context, engine storage.Engine, root *storage.URI, name string, parent ksuid.KSUID, at journal.ID) (*BranchConfig, error) {
	poolPath := p.Path(root)
	branchesPath := poolPath.AppendPath(BranchesTag)
	branches, err := kvs.Open(ctx, engine, branchesPath, branchStoreTypes)
	if err != nil {
		return nil, err
	}
	if _, err := branches.Lookup(ctx, name); err == nil {
		return nil, fmt.Errorf("%s/%s: %w", p.Name, name, ErrBranchExists)
	}
	branchRef := newBranchConfig(name, parent)
	if err := branchRef.Create(ctx, engine, poolPath, p.Layout.Order, at); err != nil {
		return nil, err
	}
	if err := branches.Insert(ctx, name, branchRef); err != nil {
		branchRef.Remove(ctx, engine, poolPath)
		return nil, err
	}
	return branchRef, err
}

func (p *PoolConfig) Open(ctx context.Context, engine storage.Engine, root *storage.URI) (*Pool, error) {
	path := p.Path(root)
	branchesPath := path.AppendPath(BranchesTag)
	types := []interface{}{BranchConfig{}}
	branches, err := kvs.Open(ctx, engine, branchesPath, types)
	if err != nil {
		return nil, err
	}
	return &Pool{
		PoolConfig:   *p,
		engine:       engine,
		Path:         path,
		DataPath:     DataPath(path),
		IndexPath:    IndexPath(path),
		BranchesPath: branchesPath,
		branches:     branches,
	}, nil
}

func (p *PoolConfig) Remove(ctx context.Context, engine storage.Engine, root *storage.URI) error {
	return engine.DeleteByPrefix(ctx, p.Path(root))
}

func (p *Pool) removeBranch(ctx context.Context, id ksuid.KSUID) error {
	branch, err := p.OpenBranchByID(ctx, id)
	if err != nil {
		return err
	}
	if err := p.branches.Delete(ctx, branch.Name, nil); err != nil {
		return err
	}
	return branch.Remove(ctx, p.engine, p.Path)
}

func (p *Pool) newScheduler(ctx context.Context, zctx *zson.Context, branchID, at ksuid.KSUID, span extent.Span, filter zbuf.Filter) (proc.Scheduler, error) {
	branch, err := p.OpenBranchByID(ctx, branchID)
	if err != nil {
		return nil, err
	}
	snap, err := branch.Snapshot(ctx, at)
	if err != nil {
		return nil, err
	}
	return NewSortedScheduler(ctx, zctx, p, snap, span, filter), nil
}

func (p *Pool) ListBranches(ctx context.Context) ([]BranchConfig, error) {
	entries, err := p.branches.All(ctx)
	if err != nil || len(entries) == 0 {
		return nil, err
	}
	branches := make([]BranchConfig, 0, len(entries))
	for _, entry := range entries {
		b, ok := entry.Value.(*BranchConfig)
		if !ok {
			return nil, fmt.Errorf("system error: unknown type found in branch config: %T", entry.Value)
		}
		branches = append(branches, *b)
	}
	return branches, nil
}

func (p *Pool) LookupBranchByName(ctx context.Context, name string) (*BranchConfig, error) {
	v, err := p.branches.Lookup(ctx, name)
	if err != nil {
		if err == kvs.ErrNoSuchKey {
			err = fmt.Errorf("%q: %w: %q", p.Name, ErrBranchNotFound, name)
		}
		return nil, err
	}
	b, ok := v.(*BranchConfig)
	if !ok {
		return nil, fmt.Errorf("system error: unknown type found in branch config: %T", v)
	}
	return b, nil
}

func (p *Pool) openBranch(ctx context.Context, branchRef *BranchConfig) (*Branch, error) {
	return branchRef.Open(ctx, p.engine, p.Path, p)
}

func (p *Pool) OpenBranchByID(ctx context.Context, id ksuid.KSUID) (*Branch, error) {
	entries, err := p.branches.All(ctx)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		branchRef, ok := entry.Value.(*BranchConfig)
		if !ok {
			return nil, fmt.Errorf("system error: branch table entry wrong type %T", entry.Value)
		}
		if branchRef.ID == id {
			return p.openBranch(ctx, branchRef)
		}
	}
	return nil, fmt.Errorf("branch id %s does not exist in pool %q", id, p.Name)
}

func (p *Pool) OpenBranchByName(ctx context.Context, name string) (*Branch, error) {
	branchRef, err := p.LookupBranchByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return p.openBranch(ctx, branchRef)
}

func (p *Pool) batchifyBranches(ctx context.Context, recs zbuf.Array, m *zson.MarshalZNGContext, f expr.Filter) (zbuf.Array, error) {
	branches, err := p.ListBranches(ctx)
	if err != nil {
		return nil, err
	}
	for _, branchRef := range branches {
		meta := BranchMeta{p.PoolConfig, branchRef}
		rec, err := m.MarshalRecord(&meta)
		if err != nil {
			return nil, err
		}
		if f == nil || f(rec) {
			recs.Append(rec)
		}
	}
	return recs, nil
}

//XXX this is inefficient but is only meant for interactive queries...?
func (p *Pool) ObjectExists(ctx context.Context, id ksuid.KSUID) (bool, error) {
	return p.engine.Exists(ctx, segment.RowObjectPath(p.DataPath, id))
}

//XXX for backward compat keep this for now, and return branchstats for pool/main
type PoolStats struct {
	Size int64 `zng:"size"`
	// XXX (nibs) - This shouldn't be a span because keys don't have to be time.
	Span *nano.Span `zng:"span"`
}

func (p *Pool) Stats(ctx context.Context, snap commit.View) (info PoolStats, err error) {
	ch := make(chan segment.Reference)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		err = ScanSpan(ctx, snap, nil, p.Layout.Order, ch)
		close(ch)
	}()
	// XXX this doesn't scale... it should be stored in the snapshot and is
	// not easy to compute in the face of deletes...
	var poolSpan *extent.Generic
	for segment := range ch {
		info.Size += segment.RowSize
		if poolSpan == nil {
			poolSpan = extent.NewGenericFromOrder(segment.First, segment.Last, p.Layout.Order)
		} else {
			poolSpan.Extend(segment.First)
			poolSpan.Extend(segment.Last)
		}
	}
	//XXX need to change API to take return key range
	if poolSpan != nil {
		min := poolSpan.First()
		if min.Type == zng.TypeTime {
			firstTs, _ := zng.DecodeTime(min.Bytes)
			lastTs, _ := zng.DecodeTime(poolSpan.Last().Bytes)
			if lastTs < firstTs {
				firstTs, lastTs = lastTs, firstTs
			}
			span := nano.NewSpanTs(firstTs, lastTs+1)
			info.Span = &span
		}
	}
	return info, err
}

func (p *Pool) Main(ctx context.Context) (BranchMeta, error) {
	branch, err := p.OpenBranchByName(ctx, "main")
	if err != nil {
		return BranchMeta{}, err
	}
	return BranchMeta{p.PoolConfig, branch.BranchConfig}, nil
}

func DataPath(poolPath *storage.URI) *storage.URI {
	return poolPath.AppendPath(DataTag)
}

func IndexPath(poolPath *storage.URI) *storage.URI {
	return poolPath.AppendPath(IndexTag)
}

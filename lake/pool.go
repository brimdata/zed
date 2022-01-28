package lake

import (
	"context"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/lake/branches"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/pools"
	"github.com/brimdata/zed/lake/tags"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

const (
	DataTag     = "data"
	IndexTag    = "index"
	BranchesTag = "branches"
	TagsTag     = "tags"
	CommitsTag  = "commits"
)

type Pool struct {
	pools.Config
	engine    storage.Engine
	Path      *storage.URI
	DataPath  *storage.URI
	IndexPath *storage.URI
	branches  *branches.Store
	tags      *tags.Store
	commits   *commits.Store
}

func CreatePool(ctx context.Context, config *pools.Config, engine storage.Engine, root *storage.URI) error {
	poolPath := config.Path(root)
	// branchesPath is the path to the kvs journal of BranchConfigs
	// for the pool while the commit log is stored in <pool-id>/<branch-id>.
	branchesPath := poolPath.AppendPath(BranchesTag)
	// create the branches journal store
	_, err := branches.CreateStore(ctx, engine, branchesPath)
	if err != nil {
		return err
	}
	// create the tags journal store
	tagsPath := poolPath.AppendPath(TagsTag)
	if _, err = tags.CreateStore(ctx, engine, tagsPath); err != nil {
		return err
	}
	// create the main branch in the branches journal store.  The parent
	// commit object of the initial main branch is ksuid.Nil.
	_, err = CreateBranch(ctx, config, engine, root, "main", ksuid.Nil)
	return err
}

func CreateBranch(ctx context.Context, poolConfig *pools.Config, engine storage.Engine, root *storage.URI, name string, parent ksuid.KSUID) (*branches.Config, error) {
	poolPath := poolConfig.Path(root)
	branchesPath := poolPath.AppendPath(BranchesTag)
	store, err := branches.OpenStore(ctx, engine, branchesPath)
	if err != nil {
		return nil, err
	}
	if _, err := store.LookupByName(ctx, name); err == nil {
		return nil, fmt.Errorf("%s/%s: %w", poolConfig.Name, name, branches.ErrExists)
	}
	branchConfig := branches.NewConfig(name, parent)
	if err := store.Add(ctx, branchConfig); err != nil {
		return nil, err
	}
	return branchConfig, err
}

func OpenPool(ctx context.Context, config *pools.Config, engine storage.Engine, root *storage.URI) (*Pool, error) {
	path := config.Path(root)
	branchesPath := path.AppendPath(BranchesTag)
	branches, err := branches.OpenStore(ctx, engine, branchesPath)
	if err != nil {
		return nil, err
	}
	tagsPath := path.AppendPath(TagsTag)
	tags, err := tags.OpenStore(ctx, engine, tagsPath)
	if err != nil {
		return nil, err
	}
	commitsPath := path.AppendPath(CommitsTag)
	commits, err := commits.OpenStore(engine, commitsPath)
	if err != nil {
		return nil, err
	}
	return &Pool{
		Config:    *config,
		engine:    engine,
		Path:      path,
		DataPath:  DataPath(path),
		IndexPath: IndexPath(path),
		branches:  branches,
		tags:      tags,
		commits:   commits,
	}, nil
}

func RemovePool(ctx context.Context, config *pools.Config, engine storage.Engine, root *storage.URI) error {
	return engine.DeleteByPrefix(ctx, config.Path(root))
}

func (p *Pool) removeBranch(ctx context.Context, name string) error {
	config, err := p.branches.LookupByName(ctx, name)
	if err != nil {
		return err
	}
	return p.branches.Remove(ctx, *config)
}

func (p *Pool) newScheduler(ctx context.Context, zctx *zed.Context, commit ksuid.KSUID, span extent.Span, filter zbuf.Filter, idx *dag.Filter) (proc.Scheduler, error) {
	snap, err := p.commits.Snapshot(ctx, commit)
	if err != nil {
		return nil, err
	}
	var idxFilter *index.Filter
	if idx != nil {
		if idxFilter, err = index.NewFilter(p.engine, p.IndexPath, idx); err != nil {
			return nil, err
		}
	}
	return NewSortedScheduler(ctx, zctx, p, snap, span, filter, idxFilter), nil
}

func CreateTag(ctx context.Context, poolConfig *pools.Config, engine storage.Engine, root *storage.URI, name string, parent ksuid.KSUID) (*tags.Config, error) {
	poolPath := poolConfig.Path(root)
	tagsPath := poolPath.AppendPath(TagsTag)
	store, err := tags.OpenStore(ctx, engine, tagsPath)
	if err != nil {
		return nil, err
	}
	if _, err := store.LookupByName(ctx, name); err == nil {
		return nil, fmt.Errorf("%s/%s: %w", poolConfig.Name, name, tags.ErrExists)
	}
	tagConfig := tags.NewConfig(name, parent)
	if err := store.Add(ctx, tagConfig); err != nil {
		return nil, err
	}
	return tagConfig, err
}

func (p *Pool) removeTag(ctx context.Context, name string) error {
	config, err := p.tags.LookupByName(ctx, name)
	if err != nil {
		return err
	}
	return p.tags.Remove(ctx, *config)
}

func (p *Pool) Snapshot(ctx context.Context, commit ksuid.KSUID) (commits.View, error) {
	return p.commits.Snapshot(ctx, commit)
}

func (p *Pool) ListBranches(ctx context.Context) ([]branches.Config, error) {
	return p.branches.All(ctx)
}

func (p *Pool) LookupBranchByName(ctx context.Context, name string) (*branches.Config, error) {
	return p.branches.LookupByName(ctx, name)
}

func (p *Pool) ListTags(ctx context.Context) ([]tags.Config, error) {
	return p.tags.All(ctx)
}

func (p *Pool) LookupTagByName(ctx context.Context, name string) (*tags.Config, error) {
	return p.tags.LookupByName(ctx, name)
}

func (p *Pool) openBranch(ctx context.Context, config *branches.Config) (*Branch, error) {
	return OpenBranch(ctx, config, p.engine, p.Path, p)
}

func (p *Pool) OpenBranchByName(ctx context.Context, name string) (*Branch, error) {
	branchRef, err := p.LookupBranchByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return p.openBranch(ctx, branchRef)
}

func (p *Pool) batchifyBranches(ctx context.Context, zctx *zed.Context, recs []zed.Value, m *zson.MarshalZNGContext, f expr.Evaluator) ([]zed.Value, error) {
	branches, err := p.ListBranches(ctx)
	if err != nil {
		return nil, err
	}
	ectx := expr.NewContext()
	for _, branchRef := range branches {
		meta := BranchMeta{p.Config, branchRef}
		rec, err := m.MarshalRecord(&meta)
		if err != nil {
			return nil, err
		}
		if filter(zctx, ectx, rec, f) {
			recs = append(recs, *rec)
		}
	}
	return recs, nil
}

func filter(zctx *zed.Context, ectx expr.Context, this *zed.Value, e expr.Evaluator) bool {
	if e == nil {
		return true
	}
	val, ok := expr.EvalBool(zctx, ectx, this, e)
	return ok && val.Bytes != nil && zed.IsTrue(val.Bytes)
}

func (p *Pool) batchifyTags(ctx context.Context, zctx *zed.Context, f expr.Evaluator) ([]zed.Value, error) {
	tags, err := p.ListTags(ctx)
	if err != nil {
		return nil, err
	}
	m := zson.NewZNGMarshalerWithContext(zctx)
	m.Decorate(zson.StylePackage)
	recs := make([]zed.Value, 0, len(tags))
	ectx := expr.NewContext()
	for _, tagRef := range tags {
		meta := TagMeta{p.Config, tagRef}
		rec, err := m.MarshalRecord(&meta)
		if err != nil {
			return nil, err
		}
		if filter(zctx, ectx, rec, f) {
			recs = append(recs, *rec)
		}
	}
	return recs, nil
}

type BranchTip struct {
	Name   string
	Commit ksuid.KSUID
}

func (p *Pool) batchifyBranchTips(ctx context.Context, zctx *zed.Context, f expr.Evaluator) ([]zed.Value, error) {
	branches, err := p.ListBranches(ctx)
	if err != nil {
		return nil, err
	}
	m := zson.NewZNGMarshalerWithContext(zctx)
	m.Decorate(zson.StylePackage)
	recs := make([]zed.Value, 0, len(branches))
	ectx := expr.NewContext()
	for _, branchRef := range branches {
		rec, err := m.MarshalRecord(&BranchTip{branchRef.Name, branchRef.Commit})
		if err != nil {
			return nil, err
		}
		if filter(zctx, ectx, rec, f) {
			recs = append(recs, *rec)
		}
	}
	return recs, nil
}

//XXX this is inefficient but is only meant for interactive queries...?
func (p *Pool) ObjectExists(ctx context.Context, id ksuid.KSUID) (bool, error) {
	return p.engine.Exists(ctx, data.RowObjectPath(p.DataPath, id))
}

//XXX for backward compat keep this for now, and return branchstats for pool/main
type PoolStats struct {
	Size int64 `zed:"size"`
	// XXX (nibs) - This shouldn't be a span because keys don't have to be time.
	Span *nano.Span `zed:"span"`
}

func (p *Pool) Stats(ctx context.Context, snap commits.View) (info PoolStats, err error) {
	ch := make(chan data.Object)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		err = ScanSpan(ctx, snap, nil, p.Layout.Order, ch)
		close(ch)
	}()
	// XXX this doesn't scale... it should be stored in the snapshot and is
	// not easy to compute in the face of deletes...
	var poolSpan *extent.Generic
	for object := range ch {
		info.Size += object.RowSize
		if poolSpan == nil {
			poolSpan = extent.NewGenericFromOrder(object.First, object.Last, p.Layout.Order)
		} else {
			poolSpan.Extend(&object.First)
			poolSpan.Extend(&object.Last)
		}
	}
	//XXX need to change API to take return key range
	if poolSpan != nil {
		min := poolSpan.First()
		if min.Type == zed.TypeTime {
			firstTs := zed.DecodeTime(min.Bytes)
			lastTs := zed.DecodeTime(poolSpan.Last().Bytes)
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
	return BranchMeta{p.Config, branch.Config}, nil
}

func DataPath(poolPath *storage.URI) *storage.URI {
	return poolPath.AppendPath(DataTag)
}

func IndexPath(poolPath *storage.URI) *storage.URI {
	return poolPath.AppendPath(IndexTag)
}

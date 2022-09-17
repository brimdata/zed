package lake

import (
	"context"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake/branches"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/lake/data"
	"github.com/brimdata/zed/lake/pools"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

const (
	DataTag     = "data"
	IndexTag    = "index"
	BranchesTag = "branches"
	CommitsTag  = "commits"
)

type Pool struct {
	pools.Config
	engine    storage.Engine
	Path      *storage.URI
	DataPath  *storage.URI
	IndexPath *storage.URI
	branches  *branches.Store
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

func (p *Pool) Snapshot(ctx context.Context, commit ksuid.KSUID) (commits.View, error) {
	return p.commits.Snapshot(ctx, commit)
}

func (p *Pool) OpenCommitLog(ctx context.Context, zctx *zed.Context, commit ksuid.KSUID) zio.Reader {
	return p.commits.OpenCommitLog(ctx, zctx, commit, ksuid.Nil)
}

func (p *Pool) OpenCommitLogAsZNG(ctx context.Context, zctx *zed.Context, commit ksuid.KSUID) (*zngio.Reader, error) {
	return p.commits.OpenAsZNG(ctx, zctx, commit, ksuid.Nil)
}

func (p *Pool) Storage() storage.Engine {
	return p.engine
}

func (p *Pool) ListBranches(ctx context.Context) ([]branches.Config, error) {
	return p.branches.All(ctx)
}

func (p *Pool) LookupBranchByName(ctx context.Context, name string) (*branches.Config, error) {
	return p.branches.LookupByName(ctx, name)
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

func (p *Pool) BatchifyBranches(ctx context.Context, zctx *zed.Context, recs []zed.Value, m *zson.MarshalZNGContext, f expr.Evaluator) ([]zed.Value, error) {
	branches, err := p.ListBranches(ctx)
	if err != nil {
		return nil, err
	}
	ectx := expr.NewContext()
	for _, branchRef := range branches {
		meta := BranchMeta{p.Config, branchRef}
		rec, err := m.Marshal(&meta)
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

type BranchTip struct {
	Name   string
	Commit ksuid.KSUID
}

func (p *Pool) BatchifyBranchTips(ctx context.Context, zctx *zed.Context, f expr.Evaluator) ([]zed.Value, error) {
	branches, err := p.ListBranches(ctx)
	if err != nil {
		return nil, err
	}
	m := zson.NewZNGMarshalerWithContext(zctx)
	m.Decorate(zson.StylePackage)
	recs := make([]zed.Value, 0, len(branches))
	ectx := expr.NewContext()
	for _, branchRef := range branches {
		rec, err := m.Marshal(&BranchTip{branchRef.Name, branchRef.Commit})
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
	return p.engine.Exists(ctx, data.SequenceURI(p.DataPath, id))
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

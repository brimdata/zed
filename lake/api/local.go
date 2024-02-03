package api

import (
	"context"
	"errors"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/runtime/exec"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
)

type local struct {
	root     *lake.Root
	compiler runtime.Compiler
}

var _ Interface = (*local)(nil)

func OpenLocalLake(ctx context.Context, logger *zap.Logger, lakePath string) (Interface, error) {
	uri, err := storage.ParseURI(lakePath)
	if err != nil {
		return nil, err
	}
	engine := storage.NewLocalEngine()
	root, err := lake.Open(ctx, engine, logger, uri)
	if err != nil {
		return nil, err
	}
	return FromRoot(root), nil
}

func CreateLocalLake(ctx context.Context, logger *zap.Logger, lakePath string) (Interface, error) {
	uri, err := storage.ParseURI(lakePath)
	if err != nil {
		return nil, err
	}
	engine := storage.NewLocalEngine()
	root, err := lake.Create(ctx, engine, logger, uri)
	if err != nil {
		return nil, err
	}
	return FromRoot(root), nil
}

func FromRoot(root *lake.Root) Interface {
	return &local{root: root, compiler: compiler.NewLakeCompiler(root)}
}

func (l *local) Root() *lake.Root {
	return l.root
}

func (l *local) CreatePool(ctx context.Context, name string, sortKey order.SortKey, seekStride int, thresh int64) (ksuid.KSUID, error) {
	if name == "" {
		return ksuid.Nil, errors.New("no pool name provided")
	}
	pool, err := l.root.CreatePool(ctx, name, sortKey, seekStride, thresh)
	if err != nil {
		return ksuid.Nil, err
	}
	return pool.ID, nil
}

func (l *local) RemovePool(ctx context.Context, id ksuid.KSUID) error {
	return l.root.RemovePool(ctx, id)

}

func (l *local) RenamePool(ctx context.Context, id ksuid.KSUID, name string) error {
	if name == "" {
		return errors.New("no pool name provided")
	}
	return l.root.RenamePool(ctx, id, name)
}

func (l *local) CreateBranch(ctx context.Context, poolID ksuid.KSUID, name string, parent ksuid.KSUID) error {
	_, err := l.root.CreateBranch(ctx, poolID, name, parent)
	return err
}

func (l *local) RemoveBranch(ctx context.Context, poolID ksuid.KSUID, branchName string) error {
	return l.root.RemoveBranch(ctx, poolID, branchName)
}

func (l *local) MergeBranch(ctx context.Context, poolID ksuid.KSUID, childBranch, parentBranch string, message api.CommitMessage) (ksuid.KSUID, error) {
	return l.root.MergeBranch(ctx, poolID, childBranch, parentBranch, message.Author, message.Body)
}

func (l *local) Compact(ctx context.Context, poolID ksuid.KSUID, branchName string, objects []ksuid.KSUID, writeVectors bool, commit api.CommitMessage) (ksuid.KSUID, error) {
	pool, err := l.root.OpenPool(ctx, poolID)
	if err != nil {
		return ksuid.Nil, err
	}
	return exec.Compact(ctx, l.root, pool, branchName, objects, writeVectors, commit.Author, commit.Body, commit.Meta)
}

func (l *local) Query(ctx context.Context, head *lakeparse.Commitish, src string, srcfiles ...string) (zio.ReadCloser, error) {
	q, err := l.QueryWithControl(ctx, head, src, srcfiles...)
	if err != nil {
		return nil, err
	}
	return zio.NewReadCloser(zbuf.NoControl(q), q), nil
}

func (l *local) QueryWithControl(ctx context.Context, head *lakeparse.Commitish, src string, srcfiles ...string) (zbuf.ProgressReadCloser, error) {
	flowgraph, err := l.compiler.Parse(src, srcfiles...)
	if err != nil {
		return nil, err
	}
	q, err := runtime.CompileLakeQuery(ctx, zed.NewContext(), l.compiler, flowgraph, head)
	if err != nil {
		return nil, err
	}
	return runtime.AsProgressReadCloser(q), nil
}

func (l *local) PoolID(ctx context.Context, poolName string) (ksuid.KSUID, error) {
	if poolName == "" {
		return ksuid.Nil, errors.New("no pool name provided")
	}
	if id, err := lakeparse.ParseID(poolName); err == nil {
		if _, err := l.root.OpenPool(ctx, id); err == nil {
			return id, nil
		}
	}
	return l.root.PoolID(ctx, poolName)
}

func (l *local) CommitObject(ctx context.Context, poolID ksuid.KSUID, branchName string) (ksuid.KSUID, error) {
	return l.root.CommitObject(ctx, poolID, branchName)
}

func (l *local) lookupBranch(ctx context.Context, poolID ksuid.KSUID, branchName string) (*lake.Pool, *lake.Branch, error) {
	pool, err := l.root.OpenPool(ctx, poolID)
	if err != nil {
		return nil, nil, err
	}
	branch, err := pool.OpenBranchByName(ctx, branchName)
	if err != nil {
		return nil, nil, err
	}
	return pool, branch, nil
}

func (l *local) Load(ctx context.Context, ztcx *zed.Context, poolID ksuid.KSUID, branchName string, r zio.Reader, message api.CommitMessage) (ksuid.KSUID, error) {
	_, branch, err := l.lookupBranch(ctx, poolID, branchName)
	if err != nil {
		return ksuid.Nil, err
	}
	return branch.Load(ctx, ztcx, r, message.Author, message.Body, message.Meta)
}

func (l *local) Delete(ctx context.Context, poolID ksuid.KSUID, branchName string, ids []ksuid.KSUID, message api.CommitMessage) (ksuid.KSUID, error) {
	_, branch, err := l.lookupBranch(ctx, poolID, branchName)
	if err != nil {
		return ksuid.Nil, err
	}
	commitID, err := branch.Delete(ctx, ids, message.Author, message.Body)
	if err != nil {
		return ksuid.Nil, err
	}
	return commitID, nil
}

func (l *local) DeleteWhere(ctx context.Context, poolID ksuid.KSUID, branchName, src string, commit api.CommitMessage) (ksuid.KSUID, error) {
	op, err := l.compiler.Parse(src)
	if err != nil {
		return ksuid.Nil, err
	}
	_, branch, err := l.lookupBranch(ctx, poolID, branchName)
	if err != nil {
		return ksuid.Nil, err
	}
	return branch.DeleteWhere(ctx, l.compiler, op, commit.Author, commit.Body, commit.Meta)
}

func (l *local) Revert(ctx context.Context, poolID ksuid.KSUID, branchName string, commitID ksuid.KSUID, message api.CommitMessage) (ksuid.KSUID, error) {
	return l.root.Revert(ctx, poolID, branchName, commitID, message.Author, message.Body)
}

func (l *local) AddVectors(ctx context.Context, pool, revision string, ids []ksuid.KSUID, message api.CommitMessage) (ksuid.KSUID, error) {
	poolID, err := l.PoolID(ctx, pool)
	if err != nil {
		return ksuid.Nil, err
	}
	_, branch, err := l.lookupBranch(ctx, poolID, revision)
	if err != nil {
		return ksuid.Nil, err
	}
	return branch.AddVectors(ctx, ids, message.Author, message.Body)
}

func (l *local) DeleteVectors(ctx context.Context, pool, revision string, ids []ksuid.KSUID, message api.CommitMessage) (ksuid.KSUID, error) {
	poolID, err := l.PoolID(ctx, pool)
	if err != nil {
		return ksuid.Nil, err
	}
	_, branch, err := l.lookupBranch(ctx, poolID, revision)
	if err != nil {
		return ksuid.Nil, err
	}
	return branch.DeleteVectors(ctx, ids, message.Author, message.Body)
}

func (l *local) Vacuum(ctx context.Context, pool, revision string, dryrun bool) ([]ksuid.KSUID, error) {
	poolID, err := l.PoolID(ctx, pool)
	if err != nil {
		return nil, err
	}
	p, err := l.root.OpenPool(ctx, poolID)
	if err != nil {
		return nil, err
	}
	commit, err := p.ResolveRevision(ctx, revision)
	if err != nil {
		return nil, err
	}
	return p.Vacuum(ctx, commit, dryrun)
}

package api

import (
	"context"
	"errors"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/rlimit"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/segmentio/ksuid"
)

type LocalSession struct {
	root   *lake.Root
	engine storage.Engine
}

var _ Interface = (*LocalSession)(nil)

func OpenLocalLake(ctx context.Context, lakePath *storage.URI) (*LocalSession, error) {
	engine := storage.NewLocalEngine()
	root, err := lake.Open(ctx, engine, lakePath)
	if err != nil {
		return nil, err
	}
	return &LocalSession{
		root:   root,
		engine: engine,
	}, nil
}

func CreateLocalLake(ctx context.Context, lakePath *storage.URI) (*LocalSession, error) {
	engine := storage.NewLocalEngine()
	root, err := lake.Create(ctx, engine, lakePath)
	if err != nil {
		return nil, err
	}
	return &LocalSession{
		root:   root,
		engine: engine,
	}, nil
}

func (l *LocalSession) CreatePool(ctx context.Context, name string, layout order.Layout, seekStride int, thresh int64) (ksuid.KSUID, error) {
	if name == "" {
		return ksuid.Nil, errors.New("no pool name provided")
	}
	pool, err := l.root.CreatePool(ctx, name, layout, seekStride, thresh)
	if err != nil {
		return ksuid.Nil, err
	}
	return pool.ID, nil
}

func (l *LocalSession) RemovePool(ctx context.Context, id ksuid.KSUID) error {
	return l.root.RemovePool(ctx, id)

}

func (l *LocalSession) RenamePool(ctx context.Context, id ksuid.KSUID, name string) error {
	if name == "" {
		return errors.New("no pool name provided")
	}
	return l.root.RenamePool(ctx, id, name)
}

func (l *LocalSession) CreateBranch(ctx context.Context, poolID ksuid.KSUID, name string, parent ksuid.KSUID) error {
	_, err := l.root.CreateBranch(ctx, poolID, name, parent)
	return err
}

func (l *LocalSession) RemoveBranch(ctx context.Context, poolID ksuid.KSUID, branchName string) error {
	return l.root.RemoveBranch(ctx, poolID, branchName)
}

func (l *LocalSession) MergeBranch(ctx context.Context, poolID ksuid.KSUID, childBranch, parentBranch string, message api.CommitMessage) (ksuid.KSUID, error) {
	return l.root.MergeBranch(ctx, poolID, childBranch, parentBranch, message.Author, message.Body)
}

func (l *LocalSession) AddIndexRules(ctx context.Context, rules []index.Rule) error {
	return l.root.AddIndexRules(ctx, rules)
}

func (l *LocalSession) DeleteIndexRules(ctx context.Context, ids []ksuid.KSUID) ([]index.Rule, error) {
	return l.root.DeleteIndexRules(ctx, ids)
}

func (l *LocalSession) Query(ctx context.Context, head *lakeparse.Commitish, src string, srcfiles ...string) (zbuf.ProgressReader, error) {
	flowgraph, err := compiler.ParseProc(src, srcfiles...)
	if err != nil {
		return nil, err
	}
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return nil, err
	}
	q, err := runtime.NewQueryOnLake(ctx, zed.NewContext(), flowgraph, l.root, head, nil)
	if err != nil {
		return nil, err
	}
	return q.AsProgressReader(), nil
}

func (l *LocalSession) PoolID(ctx context.Context, poolName string) (ksuid.KSUID, error) {
	if poolName == "" {
		return ksuid.Nil, errors.New("no pool name provided")
	}
	return l.root.PoolID(ctx, poolName)
}

func (l *LocalSession) CommitObject(ctx context.Context, poolID ksuid.KSUID, branchName string) (ksuid.KSUID, error) {
	return l.root.CommitObject(ctx, poolID, branchName)
}

func (l *LocalSession) lookupBranch(ctx context.Context, poolID ksuid.KSUID, branchName string) (*lake.Pool, *lake.Branch, error) {
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

func (l *LocalSession) Load(ctx context.Context, poolID ksuid.KSUID, branchName string, r zio.Reader, message api.CommitMessage) (ksuid.KSUID, error) {
	_, branch, err := l.lookupBranch(ctx, poolID, branchName)
	if err != nil {
		return ksuid.Nil, err
	}
	return branch.Load(ctx, r, message.Author, message.Body, message.Meta)
}

func (l *LocalSession) Delete(ctx context.Context, poolID ksuid.KSUID, branchName string, ids []ksuid.KSUID, message api.CommitMessage) (ksuid.KSUID, error) {
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

func (l *LocalSession) Revert(ctx context.Context, poolID ksuid.KSUID, branchName string, commitID ksuid.KSUID, message api.CommitMessage) (ksuid.KSUID, error) {
	return l.root.Revert(ctx, poolID, branchName, commitID, message.Author, message.Body)
}

func (l *LocalSession) ApplyIndexRules(ctx context.Context, name string, poolID ksuid.KSUID, branchName string, inTags []ksuid.KSUID) (ksuid.KSUID, error) {
	_, branch, err := l.lookupBranch(ctx, poolID, branchName)
	if err != nil {
		return ksuid.Nil, err
	}
	tags, err := branch.LookupTags(ctx, inTags)
	if err != nil {
		return ksuid.Nil, err
	}
	rules, err := l.root.LookupIndexRules(ctx, name)
	if err != nil {
		return ksuid.Nil, err
	}
	commit, err := branch.ApplyIndexRules(ctx, rules, tags)
	if err != nil {
		return ksuid.Nil, err
	}
	return commit, nil
}

func (l *LocalSession) UpdateIndex(ctx context.Context, names []string, poolID ksuid.KSUID, branchName string) (ksuid.KSUID, error) {
	_, branch, err := l.lookupBranch(ctx, poolID, branchName)
	if err != nil {
		return ksuid.Nil, err
	}
	var rules []index.Rule
	if len(names) == 0 {
		rules, err = l.root.AllIndexRules(ctx)
	} else {
		rules, err = l.root.LookupIndexRules(ctx, names...)
	}
	if err != nil {
		return ksuid.Nil, err
	}
	return branch.UpdateIndex(ctx, rules)
}

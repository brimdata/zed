package api

import (
	"context"
	"errors"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/rlimit"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
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

func (l *LocalSession) CreatePool(ctx context.Context, name string, layout order.Layout, thresh int64) (ksuid.KSUID, error) {
	if name == "" {
		return ksuid.Nil, errors.New("no pool name provided")
	}
	pool, err := l.root.CreatePool(ctx, name, layout, thresh)
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

func (l *LocalSession) CreateBranch(ctx context.Context, poolID ksuid.KSUID, name string, parent, at ksuid.KSUID) (ksuid.KSUID, error) {
	branch, err := l.root.CreateBranch(ctx, poolID, name, parent, at)
	if err != nil {
		return ksuid.Nil, err
	}
	return branch.ID, nil
}

func (l *LocalSession) RemoveBranch(ctx context.Context, poolID, branchID ksuid.KSUID) error {
	return l.root.RemoveBranch(ctx, poolID, branchID)
}

func (l *LocalSession) MergeBranch(ctx context.Context, poolID, branchID, tag ksuid.KSUID) (ksuid.KSUID, error) {
	return l.root.MergeBranch(ctx, poolID, branchID, tag)
}

func (l *LocalSession) AddIndexRules(ctx context.Context, rules []index.Rule) error {
	return l.root.AddIndexRules(ctx, rules)
}

func (l *LocalSession) DeleteIndexRules(ctx context.Context, ids []ksuid.KSUID) ([]index.Rule, error) {
	return l.root.DeleteIndexRules(ctx, ids)
}

func (l *LocalSession) Query(ctx context.Context, d driver.Driver, src string, srcfiles ...string) (zbuf.ScannerStats, error) {
	query, err := compiler.ParseProc(src, srcfiles...)
	if err != nil {
		return zbuf.ScannerStats{}, err
	}
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return zbuf.ScannerStats{}, err
	}
	return driver.RunWithLake(ctx, d, query, zson.NewContext(), l.root)
}

func (l *LocalSession) IDs(ctx context.Context, poolName, branchName string) (ksuid.KSUID, ksuid.KSUID, error) {
	if poolName == "" {
		return ksuid.Nil, ksuid.Nil, errors.New("no pool name provided")
	}
	return l.root.IDs(ctx, poolName, branchName)
}

func (l *LocalSession) lookupBranch(ctx context.Context, poolID, branchID ksuid.KSUID) (*lake.Pool, *lake.Branch, error) {
	pool, err := l.root.OpenPool(ctx, poolID)
	if err != nil {
		return nil, nil, err
	}
	branch, err := pool.OpenBranchByID(ctx, branchID)
	if err != nil {
		return nil, nil, err
	}
	return pool, branch, nil
}

func (l *LocalSession) Load(ctx context.Context, poolID, branchID ksuid.KSUID, r zio.Reader, commit api.CommitRequest) (ksuid.KSUID, error) {
	_, branch, err := l.lookupBranch(ctx, poolID, branchID)
	if err != nil {
		return ksuid.Nil, err
	}
	return branch.Load(ctx, r, commit.Date, commit.Author, commit.Message)
}

func (l *LocalSession) Delete(ctx context.Context, poolID, branchID ksuid.KSUID, tags []ksuid.KSUID, commit *api.CommitRequest) (ksuid.KSUID, error) {
	_, branch, err := l.lookupBranch(ctx, poolID, branchID)
	if err != nil {
		return ksuid.Nil, err
	}
	ids, err := branch.LookupTags(ctx, tags)
	if err != nil {
		return ksuid.Nil, err
	}
	commitID, err := branch.Delete(ctx, ids)
	if err != nil {
		return ksuid.Nil, err
	}
	return commitID, nil
}

func (l *LocalSession) ApplyIndexRules(ctx context.Context, name string, poolID, branchID ksuid.KSUID, inTags []ksuid.KSUID) (ksuid.KSUID, error) {
	_, branch, err := l.lookupBranch(ctx, poolID, branchID)
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

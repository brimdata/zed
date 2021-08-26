package api

import (
	"context"
	"errors"
	"fmt"

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

func (l *LocalSession) CreatePool(ctx context.Context, name string, layout order.Layout, thresh int64) (*lake.PoolConfig, error) {
	if name == "" {
		return nil, errors.New("no pool name provided")
	}
	pool, err := l.root.CreatePool(ctx, name, layout, thresh)
	if err != nil {
		return nil, err
	}
	return &pool.PoolConfig, nil
}

func (l *LocalSession) RenamePool(ctx context.Context, id ksuid.KSUID, name string) error {
	if name == "" {
		return errors.New("no pool name provided")
	}
	return l.root.RenamePool(ctx, id, name)
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

func (l *LocalSession) LookupPool(ctx context.Context, name string) (*lake.PoolConfig, error) {
	if name == "" {
		return nil, errors.New("no pool name provided")
	}
	pool := l.root.LookupPoolByName(ctx, name)
	if pool == nil {
		return nil, fmt.Errorf("%s: pool not found", name)
	}
	return pool, nil
}

func (l *LocalSession) RemovePool(ctx context.Context, id ksuid.KSUID) error {
	return l.root.RemovePool(ctx, id)
}

func (l *LocalSession) lookupPool(ctx context.Context, id ksuid.KSUID) (*lake.Pool, error) {
	return l.root.OpenPool(ctx, id)
}

func (l *LocalSession) Add(ctx context.Context, poolID ksuid.KSUID, r zio.Reader, commit *api.CommitRequest) (ksuid.KSUID, error) {
	pool, err := l.lookupPool(ctx, poolID)
	if err != nil {
		return ksuid.Nil, err
	}
	id, err := pool.Add(ctx, r)
	if err != nil {
		return ksuid.Nil, err
	}
	if commit != nil {
		if err := pool.Commit(ctx, id, commit.Date, commit.Author, commit.Message); err != nil {
			return ksuid.Nil, err
		}
	}
	return id, nil
}

func (l *LocalSession) Commit(ctx context.Context, poolID, id ksuid.KSUID, commit api.CommitRequest) error {
	pool, err := l.lookupPool(ctx, poolID)
	if err != nil {
		return err
	}
	return pool.Commit(ctx, id, commit.Date, commit.Author, commit.Message)
}

func (l *LocalSession) Delete(ctx context.Context, poolID ksuid.KSUID, tags []ksuid.KSUID, commit *api.CommitRequest) (ksuid.KSUID, error) {
	pool, err := l.lookupPool(ctx, poolID)
	if err != nil {
		return ksuid.Nil, err
	}
	ids, err := pool.LookupTags(ctx, tags)
	if err != nil {
		return ksuid.Nil, err
	}
	commitID, err := pool.Delete(ctx, ids)
	if err != nil {
		return ksuid.Nil, err
	}
	if commit != nil {
		if err := pool.Commit(ctx, commitID, commit.Date, commit.Author, commit.Message); err != nil {
			return ksuid.Nil, err
		}
	}
	return commitID, nil
}

func (l *LocalSession) ApplyIndexRules(ctx context.Context, name string, poolID ksuid.KSUID, inTags []ksuid.KSUID) (ksuid.KSUID, error) {
	pool, err := l.lookupPool(ctx, poolID)
	if err != nil {
		return ksuid.Nil, err
	}
	tags, err := pool.LookupTags(ctx, inTags)
	if err != nil {
		return ksuid.Nil, err
	}
	rules, err := l.root.LookupIndexRules(ctx, name)
	if err != nil {
		return ksuid.Nil, err
	}
	commit, err := pool.ApplyIndexRules(ctx, rules, tags)
	if err != nil {
		return ksuid.Nil, err
	}
	return commit, nil
}

func (l *LocalSession) Squash(ctx context.Context, poolID ksuid.KSUID, ids []ksuid.KSUID) (ksuid.KSUID, error) {
	pool, err := l.lookupPool(ctx, poolID)
	if err != nil {
		return ksuid.Nil, err
	}
	return pool.Squash(ctx, ids)
}

func (l *LocalSession) ScanStaging(ctx context.Context, poolID ksuid.KSUID, w zio.Writer, ids []ksuid.KSUID) error {
	pool, err := l.lookupPool(ctx, poolID)
	if err != nil {
		return err
	}
	return pool.ScanStaging(ctx, w, ids)
}

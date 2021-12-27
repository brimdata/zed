package api

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/pools"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/segmentio/ksuid"
)

type Interface interface {
	Query(ctx context.Context, head *lakeparse.Commitish, src string, srcfiles ...string) (zio.Reader, error)
	QueryWithControl(ctx context.Context, head *lakeparse.Commitish, src string, srcfiles ...string) (zbuf.ProgressReader, error)
	PoolID(ctx context.Context, poolName string) (ksuid.KSUID, error)
	CommitObject(ctx context.Context, poolID ksuid.KSUID, branchName string) (ksuid.KSUID, error)
	CreatePool(context.Context, string, order.Layout, int, int64) (ksuid.KSUID, error)
	RemovePool(context.Context, ksuid.KSUID) error
	RenamePool(context.Context, ksuid.KSUID, string) error
	CreateBranch(ctx context.Context, pool ksuid.KSUID, name string, parent ksuid.KSUID) error
	RemoveBranch(ctx context.Context, pool ksuid.KSUID, branchName string) error
	MergeBranch(ctx context.Context, pool ksuid.KSUID, childBranch, parentBranch string, message api.CommitMessage) (ksuid.KSUID, error)
	Load(ctx context.Context, pool ksuid.KSUID, branch string, r zio.Reader, message api.CommitMessage) (ksuid.KSUID, error)
	Delete(ctx context.Context, pool ksuid.KSUID, branchName string, tags []ksuid.KSUID, message api.CommitMessage) (ksuid.KSUID, error)
	Revert(ctx context.Context, poolID ksuid.KSUID, branch string, commitID ksuid.KSUID, commit api.CommitMessage) (ksuid.KSUID, error)
	AddIndexRules(context.Context, []index.Rule) error
	DeleteIndexRules(context.Context, []ksuid.KSUID) ([]index.Rule, error)
	ApplyIndexRules(ctx context.Context, rule string, pool ksuid.KSUID, branchName string, ids []ksuid.KSUID) (ksuid.KSUID, error)
	UpdateIndex(ctx context.Context, names []string, pool ksuid.KSUID, branchName string) (ksuid.KSUID, error)
}

func IsLakeService(u *storage.URI) bool {
	return u.Scheme == "http" || u.Scheme == "https"
}

func ScanIndexRules(ctx context.Context, api Interface) (zio.Reader, error) {
	r, err := api.Query(ctx, nil, "from :index_rules")
	if err != nil {
		return nil, err
	}
	return zbuf.NoControl(r), nil
}

func LookupPoolByName(ctx context.Context, api Interface, name string) (*pools.Config, error) {
	b := newBuffer(pools.Config{})
	zed := fmt.Sprintf("from :pools | name == '%s'", name)
	q, err := api.Query(ctx, nil, zed)
	if err != nil {
		return nil, err
	}
	if err := zio.Copy(b, zbuf.NoControl(q)); err != nil {
		return nil, err
	}
	switch len(b.results) {
	case 0:
		return nil, fmt.Errorf("%q: pool not found", name)
	case 1:
		pool, ok := b.results[0].(*pools.Config)
		if !ok {
			return nil, fmt.Errorf("internal error: pool record has wrong type: %T", b.results[0])
		}
		return pool, nil
	default:
		return nil, fmt.Errorf("internal error: multiple pools found with same name: %s", name)
	}
}

func LookupPoolByID(ctx context.Context, api Interface, id ksuid.KSUID) (*pools.Config, error) {
	b := newBuffer(pools.Config{})
	zed := fmt.Sprintf("from :pools | id == from_hex('%s')", idToHex(id))
	q, err := api.Query(ctx, nil, zed)
	if err != nil {
		return nil, err
	}
	if err := zio.Copy(b, zbuf.NoControl(q)); err != nil {
		return nil, err
	}
	switch len(b.results) {
	case 0:
		return nil, fmt.Errorf("%s: pool not found", id)
	case 1:
		pool, ok := b.results[0].(*pools.Config)
		if !ok {
			return nil, fmt.Errorf("internal error: pool record has wrong type: %T", b.results[0])
		}
		return pool, nil
	default:
		return nil, fmt.Errorf("internal error: multiple pools found with same id: %s", id)
	}
}

func LookupBranchByName(ctx context.Context, api Interface, poolName, branchName string) (*lake.BranchMeta, error) {
	b := newBuffer(lake.BranchMeta{})
	zed := fmt.Sprintf("from :branches | pool.name == '%s' branch.name == '%s'", poolName, branchName)
	q, err := api.Query(ctx, nil, zed)
	if err != nil {
		return nil, err
	}
	if err := zio.Copy(b, zbuf.NoControl(q)); err != nil {
		return nil, err
	}
	switch len(b.results) {
	case 0:
		return nil, fmt.Errorf("%q: branch not found", poolName+"/"+branchName)
	case 1:
		branch, ok := b.results[0].(*lake.BranchMeta)
		if !ok {
			return nil, fmt.Errorf("internal error: branch record has wrong type: %T", b.results[0])
		}
		return branch, nil
	default:
		return nil, fmt.Errorf("internal error: multiple branches found with same name: %s", poolName+"/"+branchName)
	}
}

func LookupBranchByID(ctx context.Context, api Interface, id ksuid.KSUID) (*lake.BranchMeta, error) {
	b := newBuffer(lake.BranchMeta{})
	zed := fmt.Sprintf("from :branches | branch.id == 'from_hex(%s)'", idToHex(id))
	q, err := api.Query(ctx, nil, zed)
	if err != nil {
		return nil, err
	}
	if err := zio.Copy(b, zbuf.NoControl(q)); err != nil {
		return nil, err
	}
	switch len(b.results) {
	case 0:
		return nil, fmt.Errorf("%s: branch not found", id)
	case 1:
		branch, ok := b.results[0].(*lake.BranchMeta)
		if !ok {
			return nil, fmt.Errorf("internal error: branch record has wrong type: %T", b.results[0])
		}
		return branch, nil
	default:
		return nil, fmt.Errorf("internal error: multiple branches found with same id: %s", id)
	}
}

func idToHex(id ksuid.KSUID) string {
	return hex.EncodeToString(id.Bytes())
}

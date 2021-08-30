package api

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/segmentio/ksuid"
)

type Interface interface {
	Query(ctx context.Context, d driver.Driver, src string, srcfiles ...string) (zbuf.ScannerStats, error)
	IDs(ctx context.Context, poolName, branchName string) (ksuid.KSUID, ksuid.KSUID, error)
	CreatePool(context.Context, string, order.Layout, int64) (ksuid.KSUID, error)
	RemovePool(context.Context, ksuid.KSUID) error
	RenamePool(context.Context, ksuid.KSUID, string) error
	CreateBranch(ctx context.Context, pool ksuid.KSUID, name string, parent, at ksuid.KSUID) (ksuid.KSUID, error)
	RemoveBranch(ctx context.Context, pool, branch ksuid.KSUID) error
	MergeBranch(ctx context.Context, pool, branch, at ksuid.KSUID) (ksuid.KSUID, error)
	Load(ctx context.Context, pool, branch ksuid.KSUID, r zio.Reader, commit api.CommitRequest) (ksuid.KSUID, error)
	Delete(ctx context.Context, pool, branch ksuid.KSUID, tags []ksuid.KSUID, commit *api.CommitRequest) (ksuid.KSUID, error)
	AddIndexRules(context.Context, []index.Rule) error
	DeleteIndexRules(context.Context, []ksuid.KSUID) ([]index.Rule, error)
	ApplyIndexRules(ctx context.Context, rule string, pool, branch ksuid.KSUID, ids []ksuid.KSUID) (ksuid.KSUID, error)
}

func ScanIndexRules(ctx context.Context, api Interface, d driver.Driver) error {
	_, err := api.Query(ctx, d, "from [index_rules]")
	return err
}

func LookupPoolByName(ctx context.Context, api Interface, name string) (*lake.PoolConfig, error) {
	d := newQueryDriver(lake.PoolConfig{})
	zed := fmt.Sprintf("from [pools] | name == '%s'", name)
	_, err := api.Query(ctx, d, zed)
	if err != nil {
		return nil, err
	}
	switch len(d.results) {
	case 0:
		return nil, fmt.Errorf("%q: pool not found", name)
	case 1:
		pool, ok := d.results[0].(*lake.PoolConfig)
		if !ok {
			return nil, fmt.Errorf("internal error: pool record has wrong type: %T", d.results[0])
		}
		return pool, nil
	default:
		return nil, fmt.Errorf("internal error: multiple pools found with same name: %s", name)
	}
}

func LookupPoolByID(ctx context.Context, api Interface, id ksuid.KSUID) (*lake.PoolConfig, error) {
	d := newQueryDriver(lake.PoolConfig{})
	zed := fmt.Sprintf("from [pools] | id == from_hex('%s')", idToHex(id))
	_, err := api.Query(ctx, d, zed)
	if err != nil {
		return nil, err
	}
	switch len(d.results) {
	case 0:
		return nil, fmt.Errorf("%s: pool not found", id)
	case 1:
		pool, ok := d.results[0].(*lake.PoolConfig)
		if !ok {
			return nil, fmt.Errorf("internal error: pool record has wrong type: %T", d.results[0])
		}
		return pool, nil
	default:
		return nil, fmt.Errorf("internal error: multiple pools found with same id: %s", id)
	}
}

func LookupBranchByName(ctx context.Context, api Interface, poolName, branchName string) (*lake.BranchMeta, error) {
	d := newQueryDriver(lake.BranchMeta{})
	zed := fmt.Sprintf("from [branches] | pool.name == '%s' branch.name == '%s'", poolName, branchName)
	_, err := api.Query(ctx, d, zed)
	if err != nil {
		return nil, err
	}
	switch len(d.results) {
	case 0:
		return nil, fmt.Errorf("%q: branch not found", poolName+"/"+branchName)
	case 1:
		branch, ok := d.results[0].(*lake.BranchMeta)
		if !ok {
			return nil, fmt.Errorf("internal error: branch record has wrong type: %T", d.results[0])
		}
		return branch, nil
	default:
		return nil, fmt.Errorf("internal error: multiple branches found with same name: %s", poolName+"/"+branchName)
	}
}

func LookupBranchByID(ctx context.Context, api Interface, id ksuid.KSUID) (*lake.BranchMeta, error) {
	d := newQueryDriver(lake.BranchMeta{})
	zed := fmt.Sprintf("from [branches] | branch.id == 'from_hex(%s)'", idToHex(id))
	_, err := api.Query(ctx, d, zed)
	if err != nil {
		return nil, err
	}
	switch len(d.results) {
	case 0:
		return nil, fmt.Errorf("%s: branch not found", id)
	case 1:
		branch, ok := d.results[0].(*lake.BranchMeta)
		if !ok {
			return nil, fmt.Errorf("internal error: branch record has wrong type: %T", d.results[0])
		}
		return branch, nil
	default:
		return nil, fmt.Errorf("internal error: multiple branches found with same id: %s", id)
	}
}

func idToHex(id ksuid.KSUID) string {
	return hex.EncodeToString(id.Bytes())
}

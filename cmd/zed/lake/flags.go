package lake

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/storage"
)

const RootEnv = "ZED_LAKE_ROOT"

func DefaultRoot() string {
	return os.Getenv(RootEnv)
}

// Flags implements flags used by all "zed lake" commands.
type Flags struct {
	Root     string
	PoolName string
	Quiet    bool
}

func (f *Flags) SetFlags(set *flag.FlagSet) {
	set.StringVar(&f.Root, "R", DefaultRoot(), "URI of path to Zed lake store")
	set.StringVar(&f.PoolName, "p", "", "name of pool")
	set.BoolVar(&f.Quiet, "q", false, "quiet mode")
}

func (f *Flags) RootPath() (*storage.URI, error) {
	return storage.ParseURI(f.Root)
}

func (f *Flags) Create(ctx context.Context, engine storage.Engine) (*lake.Root, error) {
	root, err := f.RootPath()
	if err != nil {
		return nil, err
	}
	return lake.Create(ctx, engine, root)
}

func (f *Flags) Open(ctx context.Context, engine storage.Engine) (*lake.Root, error) {
	root, err := f.RootPath()
	if err != nil {
		return nil, err
	}
	if root.Path == "" {
		return nil, errors.New("no lake path specied: use -R or ZED_LAKE_ROOT environment")
	}
	return lake.Open(ctx, engine, root)
}

func (f *Flags) OpenPool(ctx context.Context, engine storage.Engine) (*lake.Pool, error) {
	if f.PoolName == "" {
		return nil, errors.New("no pool name provided")
	}
	lk, err := f.Open(ctx, engine)
	if err != nil {
		return nil, err
	}
	pool := lk.LookupPoolByName(ctx, f.PoolName)
	if pool == nil {
		return nil, fmt.Errorf("%s: pool not found", f.PoolName)
	}
	return lk.OpenPool(ctx, pool.ID)
}

func (f *Flags) CreatePool(ctx context.Context, engine storage.Engine, layout order.Layout, thresh int64) (*lake.Pool, error) {
	if f.PoolName == "" {
		return nil, errors.New("no pool name provided")
	}
	lk, err := f.Open(ctx, engine)
	if err != nil {
		return nil, err
	}
	return lk.CreatePool(ctx, f.PoolName, layout, thresh)
}

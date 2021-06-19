package lakecli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/rlimit"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

const RootEnv = "ZED_LAKE_ROOT"

func DefaultRoot() string {
	return os.Getenv(RootEnv)
}

type LocalFlags struct {
	baseFlags
	engine storage.Engine
	root   string
}

func NewLocalFlags(set *flag.FlagSet) *LocalFlags {
	l := new(LocalFlags)
	set.StringVar(&l.root, "R", DefaultRoot(), "URI of path to Zed lake store")
	l.baseFlags.SetFlags(set)
	l.engine = storage.NewLocalEngine()
	return l
}

func (l *LocalFlags) RootPath() (*storage.URI, error) {
	if l.root == "" {
		return nil, errors.New("no lake path specied: use -R or set ZED_LAKE_ROOT")
	}
	return storage.ParseURI(l.root)
}

func (l *LocalFlags) Create(ctx context.Context) (Root, error) {
	path, err := l.RootPath()
	if err != nil {
		return nil, err
	}
	root, err := lake.Create(ctx, l.engine, path)
	if err != nil {
		return nil, err
	}
	return newLocal(root), nil
}

func (l *LocalFlags) Open(ctx context.Context) (Root, error) {
	path, err := l.RootPath()
	if err != nil {
		return nil, err
	}
	root, err := lake.Open(ctx, l.engine, path)
	if err != nil {
		return nil, err
	}
	return newLocal(root), nil
}

func (l LocalFlags) OpenPool(ctx context.Context) (Pool, error) {
	if l.poolName == "" {
		return nil, errors.New("no pool name provided")
	}
	root, err := l.Open(ctx)
	if err != nil {
		return nil, err
	}
	pool, err := root.LookupPoolByName(ctx, l.poolName)
	if pool == nil {
		return nil, fmt.Errorf("%s: pool not found", l.poolName)
	}
	if err != nil {
		return nil, err
	}
	return root.OpenPool(ctx, pool.ID)
}

func (l LocalFlags) CreatePool(ctx context.Context, layout order.Layout, thresh int64) (Pool, error) {
	if l.poolName == "" {
		return nil, errors.New("no pool name provided")
	}
	root, err := l.Open(ctx)
	if err != nil {
		return nil, err
	}
	return root.CreatePool(ctx, l.poolName, layout, thresh)
}

func (l *LocalFlags) SetRoot(s string) {
	l.root = s
}

// Create(ctx context.Context) (Root, error)
// OpenPool(ctx context.Context) (Pool, error)

type LocalRoot struct {
	*lake.Root
}

func newLocal(r *lake.Root) Root {
	return &LocalRoot{r}
}

func (r *LocalRoot) CreatePool(ctx context.Context, name string, layout order.Layout, thresh int64) (Pool, error) {
	pool, err := r.Root.CreatePool(ctx, name, layout, thresh)
	if err != nil {
		return nil, err
	}
	return &LocalPool{pool}, nil
}

func (r *LocalRoot) OpenPool(ctx context.Context, id ksuid.KSUID) (Pool, error) {
	pool, err := r.Root.OpenPool(ctx, id)
	if err != nil {
		return nil, err
	}
	return &LocalPool{pool}, nil
}

func (r *LocalRoot) ScanIndex(ctx context.Context, w zio.Writer, ids []ksuid.KSUID) error {
	if ids == nil {
		ids = r.Root.ListIndexIDs()
	}
	return r.Root.ScanIndex(ctx, w, ids)
}

func (r *LocalRoot) Query(ctx context.Context, d driver.Driver, zedSrc string) (zbuf.ScannerStats, error) {
	query, err := compiler.ParseProc(zedSrc)
	if err != nil {
		return zbuf.ScannerStats{}, fmt.Errorf("parse error: %w", err)
	}
	if _, err := rlimit.RaiseOpenFilesLimit(); err != nil {
		return zbuf.ScannerStats{}, err
	}
	return driver.RunWithLake(ctx, d, query, zson.NewContext(), r.Root)
}

func (r *LocalRoot) LookupPool(ctx context.Context, id ksuid.KSUID) (*lake.PoolConfig, error) {
	return r.Root.LookupPool(ctx, id), nil
}

func (r *LocalRoot) LookupPoolByName(ctx context.Context, name string) (*lake.PoolConfig, error) {
	return r.Root.LookupPoolByName(ctx, name), nil
}

var _ Pool = &LocalPool{}

type LocalPool struct {
	pool *lake.Pool
}

func (p *LocalPool) Config() lake.PoolConfig {
	return p.pool.PoolConfig
}

func (p *LocalPool) Add(ctx context.Context, r zio.Reader, commit *api.CommitRequest) (ksuid.KSUID, error) {
	id, err := p.pool.Add(ctx, r)
	if err != nil {
		return ksuid.Nil, err
	}
	if commit != nil {
		if err := p.pool.Commit(ctx, id, commit.Date, commit.Author, commit.Message); err != nil {
			return ksuid.Nil, err
		}
	}
	return id, nil
}

func (p *LocalPool) Commit(ctx context.Context, id ksuid.KSUID, commit api.CommitRequest) error {
	return p.pool.Commit(ctx, id, commit.Date, commit.Author, commit.Message)
}

func (p *LocalPool) Delete(ctx context.Context, tags []ksuid.KSUID, commit *api.CommitRequest) (ksuid.KSUID, error) {
	pool := p.pool
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

func (p *LocalPool) Index(ctx context.Context, indices []index.Index, tags []ksuid.KSUID) (ksuid.KSUID, error) {
	tags, err := p.pool.LookupTags(ctx, tags)
	if err != nil {
		return ksuid.Nil, err
	}
	commit, err := p.pool.Index(ctx, indices, tags)
	if err != nil {
		return ksuid.Nil, err
	}
	return commit, nil
}

func (p *LocalPool) Squash(ctx context.Context, ids []ksuid.KSUID) (ksuid.KSUID, error) {
	return p.pool.Squash(ctx, ids)
}

func (p *LocalPool) ScanStaging(ctx context.Context, w zio.Writer, ids []ksuid.KSUID) error {
	return p.pool.ScanStaging(ctx, w, ids)
}

func (p *LocalPool) ScanLog(ctx context.Context, w zio.Writer, head, tail journal.ID) error {
	r, err := p.pool.Log().OpenAsZNG(ctx, head, tail)
	if err != nil {
		return err
	}
	return zio.CopyWithContext(ctx, w, r)
}

func (p *LocalPool) ScanSegments(ctx context.Context, w zio.Writer, at string, partition bool, span extent.Span) (err error) {
	var id journal.ID
	if at != "" {
		id, err = parseJournalID(ctx, p.pool, at)
		if err != nil {
			return err
		}
	}
	snap, err := p.pool.Log().Snapshot(ctx, id)
	if err != nil {
		return err
	}
	if partition {
		return p.pool.ScanPartitions(ctx, w, snap, span)
	}
	return p.pool.ScanSegments(ctx, w, snap, span)
}

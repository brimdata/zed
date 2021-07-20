package api

import (
	"context"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/segmentio/ksuid"
)

type Interface interface {
	Query(ctx context.Context, d driver.Driver, src string, filenames ...string) (zbuf.ScannerStats, error)

	CreatePool(context.Context, string, order.Layout, int64) (*lake.PoolConfig, error)
	RemovePool(context.Context, ksuid.KSUID) error

	AddIndexRules(context.Context, []index.Rule) error
	DeleteIndexRules(context.Context, []ksuid.KSUID) ([]index.Rule, error)
	//XXX these should query zng and filter and be a function in this package
	LookupIndexRules(context.Context, string) ([]index.Rule, error)

	//XXX these should query zng and filter and be a function in this package
	LookupPoolByName(context.Context, string) (*lake.PoolConfig, error)

	// Data operations
	Add(ctx context.Context, pool ksuid.KSUID, r zio.Reader, commit *api.CommitRequest) (ksuid.KSUID, error)
	Delete(ctx context.Context, pool ksuid.KSUID, tags []ksuid.KSUID, commit *api.CommitRequest) (ksuid.KSUID, error)
	Commit(ctx context.Context, pool ksuid.KSUID, id ksuid.KSUID, commit api.CommitRequest) error
	Squash(ctx context.Context, pool ksuid.KSUID, ids []ksuid.KSUID) (ksuid.KSUID, error)
	//XXX should ref rules by name?
	ApplyIndexRules(ctx context.Context, rule string, pool ksuid.KSUID, ids []ksuid.KSUID) (ksuid.KSUID, error)

	// These should all be query endpoints... this way when the log converts to
	// a sub-pool the API here is the same...
	ScanLog(ctx context.Context, pool ksuid.KSUID, w zio.Writer, head, tail journal.ID) error
	ScanStaging(ctx context.Context, pool ksuid.KSUID, w zio.Writer, ids []ksuid.KSUID) error
	ScanSegments(ctx context.Context, pool ksuid.KSUID, w zio.Writer, at ksuid.KSUID, partitions bool, span extent.Span) error
	ScanIndexRules(ctx context.Context, w zio.Writer, ids []ksuid.KSUID) error
	ScanPools(context.Context, zio.Writer) error
}

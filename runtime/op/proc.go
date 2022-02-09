package op

import (
	"context"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/zbuf"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
)

const BatchLen = 100

type DataAdaptor interface {
	PoolID(context.Context, string) (ksuid.KSUID, error)
	CommitObject(context.Context, ksuid.KSUID, string) (ksuid.KSUID, error)
	Layout(context.Context, dag.Source) order.Layout
	NewScheduler(context.Context, *zed.Context, dag.Source, extent.Span, zbuf.Filter, *dag.Filter) (Scheduler, error)
	Open(context.Context, *zed.Context, string, zbuf.Filter) (zbuf.PullerCloser, error)
}

type Scheduler interface {
	PullScanTask() (zbuf.PullerCloser, error)
	Progress() zbuf.Progress
}

// Result is a convenient way to bundle the result of Proc.Pull() to
// send over channels.
type Result struct {
	Batch zbuf.Batch
	Err   error
}

// Context provides states used by all procs to provide the outside context
// in which they are running.
type Context struct {
	context.Context
	Logger *zap.Logger
	Zctx   *zed.Context
	cancel context.CancelFunc
}

func NewContext(ctx context.Context, zctx *zed.Context, logger *zap.Logger) *Context {
	ctx, cancel := context.WithCancel(ctx)
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Context{
		Context: ctx,
		cancel:  cancel,
		Logger:  logger,
		Zctx:    zctx,
	}
}

func DefaultContext() *Context {
	return NewContext(context.Background(), zed.NewContext(), nil)
}

func (c *Context) Cancel() {
	c.cancel()
}

func NopDone(puller zbuf.Puller) *done {
	return &done{puller}
}

type done struct {
	zbuf.Puller
}

func (*done) Done() {}

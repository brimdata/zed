package proc

import (
	"context"

	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
)

const BatchLen = 100

// proc.Interface is the interface to objects that operate on Batches of zbuf.Records
// and are arranged into a flowgraph to perform pattern matching and analytics.
// A proc is generally single-threaded unless lengths are taken to implement
// concurrency within a Proc.  The model is receiver-driven, stream-oriented
// data processing.  Downstream procs Pull() batches of data from upstream procs.
// Normally, a proc pulls data until end of stream (nil batch and nil error)
// or error (nil batch and non-nil error).  If a proc wants to end before
// end of stream, it calls the Done() method on its parent.  A proc implementation
// may assume calls to Pull() and Done() are single threaded so any arrangement
// of calls to Pull() and Done() cannot be done concurrently.  In short, never
// call Done() concurrently to another goroutine calling Pull().
type Interface interface {
	zbuf.Puller
	Done()
}

type DataAdaptor interface {
	Lookup(context.Context, string) (ksuid.KSUID, error)
	Layout(context.Context, ksuid.KSUID) (order.Layout, error)
	NewScheduler(context.Context, *zson.Context, ksuid.KSUID, ksuid.KSUID, extent.Span, zbuf.Filter) (Scheduler, error)
	Open(context.Context, *zson.Context, string, zbuf.Filter) (zbuf.PullerCloser, error)
	Get(context.Context, *zson.Context, string, zbuf.Filter) (zbuf.PullerCloser, error)
}

type Scheduler interface {
	PullScanTask() (zbuf.PullerCloser, error)
	Stats() zbuf.ScannerStats
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
	Logger   *zap.Logger
	Warnings chan string
	Zctx     *zson.Context
	cancel   context.CancelFunc
}

func NewContext(ctx context.Context, zctx *zson.Context, logger *zap.Logger) *Context {
	ctx, cancel := context.WithCancel(ctx)
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Context{
		Context:  ctx,
		cancel:   cancel,
		Logger:   logger,
		Warnings: make(chan string, 5),
		Zctx:     zctx,
	}
}

func DefaultContext() *Context {
	return NewContext(context.Background(), zson.NewContext(), nil)
}

func (c *Context) Cancel() {
	c.cancel()
}

func EOS(batch zbuf.Batch, err error) bool {
	return batch == nil || err != nil
}

func NopDone(puller zbuf.Puller) *done {
	return &done{puller}
}

type done struct {
	zbuf.Puller
}

func (*done) Done() {}

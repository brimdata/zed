package op

import (
	"context"
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zbuf"
	"go.uber.org/zap"
)

const BatchLen = 100

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
	// WaitGroup is used to ensure that goroutines complete cleanup work
	// (e.g., removing temporary files) before Cancel returns.
	WaitGroup sync.WaitGroup
	Zctx      *zed.Context
	cancel    context.CancelFunc
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

// Cancel cancels the context.  Cancel must be called to ensure that operators
// complete cleanup work (e.g., removing temporary files).
func (c *Context) Cancel() {
	c.cancel()
	c.WaitGroup.Wait()
}

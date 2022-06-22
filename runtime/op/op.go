package op

import (
	"context"

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

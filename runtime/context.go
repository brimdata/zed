package runtime

import (
	"context"
	"sync"

	"github.com/brimdata/zed"
)

// Context provides states used by all procs to provide the outside context
// in which they are running.
type Context struct {
	context.Context
	// WaitGroup is used to ensure that goroutines complete cleanup work
	// (e.g., removing temporary files) before Cancel returns.
	WaitGroup sync.WaitGroup
	Zctx      *zed.Context
	cancel    context.CancelFunc
}

func NewContext(ctx context.Context, zctx *zed.Context) *Context {
	ctx, cancel := context.WithCancel(ctx)
	return &Context{
		Context: ctx,
		cancel:  cancel,
		Zctx:    zctx,
	}
}

func DefaultContext() *Context {
	return NewContext(context.Background(), zed.NewContext())
}

// Cancel cancels the context.  Cancel must be called to ensure that operators
// complete cleanup work (e.g., removing temporary files).
func (c *Context) Cancel() {
	c.cancel()
	c.WaitGroup.Wait()
}

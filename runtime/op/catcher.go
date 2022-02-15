package op

import (
	"fmt"
	"runtime/debug"

	"github.com/brimdata/zed/zbuf"
)

// Catcher wraps an Interface with a Pull method that recovers panics
// and turns them into errors.  It should be wrapped around the output puller
// of a flowgraph and the top-level puller of any goroutine created inside
// of a flowgraph.
type Catcher struct {
	parent zbuf.Puller
}

func NewCatcher(parent zbuf.Puller) *Catcher {
	return &Catcher{parent}
}

func (c *Catcher) Pull(done bool) (b zbuf.Batch, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %+v\n%s\n", r, string(debug.Stack()))
		}
	}()
	return c.parent.Pull(done)
}

package proc

import (
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zqe"
)

type Catcher struct {
	parent Interface
}

func NewCatcher(parent Interface) *Catcher {
	return &Catcher{parent}
}

// SafePull runs a Pull and catches panics and turns them out errors.
// This should be called out the output puller of a flowgraph and by
// the top-level puller of all new goroutine created inside of a flowgraph.
func (c *Catcher) Pull() (b zbuf.Batch, err error) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		err = zqe.RecoverError(r)
	}()
	b, err = c.parent.Pull()
	return
}

func (c *Catcher) Done() {
	c.parent.Done()
}

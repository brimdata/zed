package pass

import (
	"github.com/brimdata/zed/zbuf"
)

type Proc struct {
	parent zbuf.Puller
}

func New(parent zbuf.Puller) *Proc {
	return &Proc{parent}
}

func (p *Proc) Pull(done bool) (zbuf.Batch, error) {
	return p.parent.Pull(done)
}

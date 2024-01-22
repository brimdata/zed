package pass

import (
	"github.com/brimdata/zed/zbuf"
)

type Op struct {
	parent zbuf.Puller
}

func New(parent zbuf.Puller) *Op {
	return &Op{parent}
}

func (o *Op) Pull(done bool) (zbuf.Batch, error) {
	return o.parent.Pull(done)
}

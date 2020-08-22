package pass

import (
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
)

type Proc struct {
	proc.Parent
}

func New(parent proc.Parent) *Proc {
	return &Proc{parent}
}

func (p *Proc) Pull() (zbuf.Batch, error) {
	return p.Get()
}

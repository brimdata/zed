package pass

import (
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
)

type Proc struct {
	proc.Parent
}

func New(parent proc.Interface) *Proc {
	return &Proc{
		Parent: proc.Parent{parent},
	}
}

func (p *Proc) Pull() (zbuf.Batch, error) {
	return p.Parent.Pull()
}

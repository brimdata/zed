package proc

import "github.com/brimsec/zq/zbuf"

type Pass struct {
	Base
}

func NewPass(c *Context, parent Proc) *Pass {
	return &Pass{Base{Context: c, Parent: parent}}
}

func (p *Pass) Pull() (zbuf.Batch, error) {
	return p.Get()
}

package proc

import "github.com/mccanne/zq/pkg/zson"

type Pass struct {
	Base
}

func NewPass(c *Context, parent Proc) *Pass {
	return &Pass{Base{Context: c, Parent: parent}}
}

func (p *Pass) Pull() (zson.Batch, error) {
	return p.Get()
}

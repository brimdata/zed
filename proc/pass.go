package proc

import "github.com/mccanne/zq/pkg/zson"

type PassProc struct {
	Base
}

func NewPassProc(c *Context, parent Proc) *PassProc {
	return &PassProc{Base{Context: c, Parent: parent}}
}

func (p *PassProc) Pull() (zson.Batch, error) {
	return p.Get()
}

package expr

import (
	"github.com/brimdata/zed/vector"
)

type Evaluator interface {
	Eval(vector.Any) vector.Any
}

type Function interface {
	Call([]vector.Any) vector.Any
}

type Call struct {
	fn    Function
	exprs []Evaluator
	args  []vector.Any
}

func NewCall(fn Function, exprs []Evaluator) *Call {
	return &Call{
		fn:    fn,
		exprs: exprs,
		args:  make([]vector.Any, len(exprs)),
	}
}

func (c *Call) Eval(this vector.Any) vector.Any {
	for k, e := range c.exprs {
		c.args[k] = e.Eval(this)
	}
	return c.fn.Call(c.args)
}

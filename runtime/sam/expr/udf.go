package expr

import (
	"slices"

	"github.com/brimdata/zed"
)

const maxStackDepth = 10_000

type UDF struct {
	Body Evaluator
}

func (u *UDF) Call(ectx Context, args []zed.Value) zed.Value {
	stack := 1
	if f, ok := ectx.(*frame); ok {
		stack += f.stack
	}
	if stack > maxStackDepth {
		panic("stack overflow")
	}
	// args must be cloned otherwise the values will be overwritten in
	// recursive calls.
	f := &frame{ectx, stack, slices.Clone(args)}
	defer f.exit()
	return u.Body.Eval(f, zed.Null)
}

type frame struct {
	Context
	stack int
	vars  []zed.Value
}

func (f *frame) Vars() []zed.Value {
	return f.vars
}

func (f *frame) exit() {
	f.stack--
}

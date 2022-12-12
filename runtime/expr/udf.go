package expr

import (
	"github.com/brimdata/zed"
	"golang.org/x/exp/slices"
)

const maxStackDepth = 10_000

type UDF struct {
	Body Evaluator
}

func (u *UDF) Call(ectx zed.Allocator, args []zed.Value) *zed.Value {
	stack := 1
	if f, ok := ectx.(*frame); ok {
		stack += f.stack
	}
	if stack > maxStackDepth {
		panic("stack overflow")
	}
	// args must be cloned otherwise the values will be overwritten in
	// recursive calls.
	f := &frame{stack: stack, vars: slices.Clone(args)}
	defer f.exit()
	return u.Body.Eval(f, zed.Null)
}

type frame struct {
	allocator
	stack int
	vars  []zed.Value
}

func (f *frame) Vars() []zed.Value {
	return f.vars
}

func (f *frame) exit() {
	f.stack--
}

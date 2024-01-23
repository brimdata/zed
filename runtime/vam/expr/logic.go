package expr

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/vector"
)

type Not struct {
	zctx *zed.Context
	expr Evaluator
}

var _ Evaluator = (*Not)(nil)

func NewLogicalNot(zctx *zed.Context, e Evaluator) *Not {
	return &Not{zctx, e}
}

func (n *Not) Eval(val vector.Any) vector.Any {
	val, ok := EvalBool(n.zctx, val, n.expr)
	if !ok {
		return val
	}
	if val.Bool() {
		return zed.False
	}
	return zed.True
}

type And struct {
	zctx *zed.Context
	lhs  Evaluator
	rhs  Evaluator
}

func NewLogicalAnd(zctx *zed.Context, lhs, rhs Evaluator) *And {
	return &And{zctx, lhs, rhs}
}

type Or struct {
	zctx *zed.Context
	lhs  Evaluator
	rhs  Evaluator
}

func NewLogicalOr(zctx *zed.Context, lhs, rhs Evaluator) *Or {
	return &Or{zctx, lhs, rhs}
}

//XXX hmm we could have a mixture of errors and normal values that
// shouldn't be carried in a union because we don't want the union
// type to bubble up to the top-level type, e.g., some values of
// a boolean comparison have divide-by-zero and thus error values
// while other values are normmal boolean results.  Maybe we can
// encode this condition as a special variant then establish the
// variation at the top-level.  Then the semantics between vam
// and sam will be the same.  Perhaps more cleanly the error
// condition can be return as a second return val of Eval since
// then we can easily wrap the errors from stage to stage also
// consitent with sam wrapping semantics.

// EvalBool evaluates e with this and if the result is a Zed bool, returns the
// result and true.  Otherwise, a Zed error (inclusive of missing) and false
// are returned.
func EvalBool(zctx *zed.Context, in vector.Any, e Evaluator) (vector.Any, bool) {
	val := e.Eval(in)
	if val.Type() == zed.TypeBool {
		return val, true
	}
	if val.IsError() {
		return val, false
	}
	return zctx.WrapError("not type bool", val), false
}

func (a *And) Eval(ectx Context, this zed.Value) zed.Value {
	lhs, ok := EvalBool(a.zctx, ectx, this, a.lhs)
	if !ok {
		return lhs
	}
	if !lhs.Bool() {
		return zed.False
	}
	rhs, ok := EvalBool(a.zctx, ectx, this, a.rhs)
	if !ok {
		return rhs
	}
	if !rhs.Bool() {
		return zed.False
	}
	return zed.True
}

func (o *Or) Eval(ectx Context, this zed.Value) zed.Value {
	lhs, ok := EvalBool(o.zctx, ectx, this, o.lhs)
	if ok && lhs.Bool() {
		return zed.True
	}
	if lhs.IsError() && !lhs.IsMissing() {
		return lhs
	}
	rhs, ok := EvalBool(o.zctx, ectx, this, o.rhs)
	if ok {
		if rhs.Bool() {
			return zed.True
		}
		return zed.False
	}
	return rhs
}

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

func (n *Not) Eval(val vector.Any) (vector.Any, vector.Any) {
	b, err := EvalBool(n.zctx, val, n.expr)
	if b == nil {
		return nil, err
	}
	bits := make([]uint64, len(b.Bits))
	for k := range bits {
		bits[k] = b.Bits[k]
	}
	return b.CopyWithBits(bits), nil
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

func (a *And) Eval(val vector.Any) (vector.Any, vector.Any) {
	lhs, err := EvalBool(a.zctx, val, a.lhs)
	if lhs == nil {
		return lhs, err
	}
	rhs, err := EvalBool(a.zctx, val, a.rhs)
	if rhs == nil {
		return rhs, err // XXX mix with lhs err
	}
	bits := make([]uint64, len(lhs.Bits))
	if len(lhs.Bits) != len(rhs.Bits) {
		panic("length mistmatch")
	}
	for k := range bits {
		bits[k] = lhs.Bits[k] & rhs.Bits[k]
	}
	//XXX intersect nulls
	return lhs.CopyWithBits(bits), nil
}

func (o *Or) Eval(val vector.Any) (vector.Any, vector.Any) {
	lhs, err := EvalBool(o.zctx, val, o.lhs)
	if lhs == nil {
		return lhs, err
	}
	rhs, err := EvalBool(o.zctx, val, o.rhs)
	if rhs == nil {
		return rhs, err // XXX mix with lhs err
	}
	bits := make([]uint64, len(lhs.Bits))
	if len(lhs.Bits) != len(rhs.Bits) {
		panic("length mistmatch")
	}
	for k := range bits {
		bits[k] = lhs.Bits[k] | rhs.Bits[k]
	}
	//XXX intersect nulls
	return lhs.CopyWithBits(bits), nil
}

// EvalBool evaluates e using val to computs a boolean result.  For elemtents
// of the result that are not boolean, an error is calculated for each non-bool
// slot and they are returned as an error.  If all of the value slots are errors,
// then the return value is nil.
func EvalBool(zctx *zed.Context, val vector.Any, e Evaluator) (*vector.Bool, vector.Any) {
	val, err := e.Eval(val)
	if val == nil {
		return nil, err
	}
	if val, ok := vector.Under(val).(*vector.Bool); ok {
		return val, err
	}
	//XXX need to implement vector.Collection and check for that here (i.e., sparse variant)
	// for now, if the vector is not uniformly boolean, we return error.
	// XXX example is a field ref a union of structs where the type of
	// the referenced field changes... there can be an arbitrary number
	// of underlying types though any given slot has only one type
	// obviously at any given time.
	return nil, vector.NewStringError(zctx, "not type bool", val.Len())
}

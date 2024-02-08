package expr

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/sam/expr/coerce"
	"github.com/brimdata/zed/vector"
)

type Compare struct {
	zctx *zed.Context
	op   string
	lhs  Evaluator
	rhs  Evaluator
}

func NewCompare(zctx *zed.Context, lhs, rhs Evaluator, operator string) *Compare {
	return &Compare{zctx, operator, lhs, rhs}
}

func (c *Compare) Eval(val vector.Any) vector.Any {
	l := vector.Under(c.lhs.Eval(val))
	r := vector.Under(c.rhs.Eval(val))
	lhs, rhs, _ := coerceVals(c.zctx, l, r)
	op := c.op
	switch op {
	case ">=":
		op = "<="
		lhs, rhs = rhs, lhs
	case ">":
		op = "<"
		lhs, rhs = rhs, lhs
	}
	l, lx := derefView(lhs)
	r, rx := derefView(rhs)
	//XXX need to handle overflow (see sam)
	//XXX unions and variants and single-value-with-error variant
	//XXX nulls... for primitives we just do the compare but we need
	// to or the nulls together
	id := lhs.Type().ID()
	switch {
	case zed.IsFloat(id):
		return compareFloats(op, l, r, lx, rx)
	case zed.IsSigned(id):
		return compareInts(op, l, r, lx, rx)
	case zed.IsUnsigned(id):
		return compareUints(op, l, r, lx, rx)
	default:
		//XXX incompatible types
		return vector.NewStringError(c.zctx, coerce.IncompatibleTypes.Error(), lhs.Len())
	}
}

func derefView(vals vector.Any) (vector.Any, []uint32) {
	var idx []uint32
	for {
		switch v := vals.(type) {
		case *vector.Dict:
			vals = v.Any
		case *vector.View:
			idx = v.Index
			vals = v.Any
		default:
			return vals, idx
		}
	}
}

func swapOp(op string) string {
	switch op {
	case "<":
		return ">"
	case "<=":
		return ">="
	case ">":
		return "<"
	case ">=":
		return "<="
	default:
		return op

	}
}

func compareFloats(op string, lhs, rhs vector.Any, lx, rx []uint32) vector.Any {
	if lc, ok := lhs.(*vector.Const); ok {
		if rc, ok := rhs.(*vector.Const); ok {
			return compareFloatConsts(op, lc, rc)
		}
		return compareFloatConst(swapOp(op), rhs.(*vector.Float), rx, lc)
	}
	if rc, ok := rhs.(*vector.Const); ok {
		return compareFloatConst(op, lhs.(*vector.Float), lx, rc)
	}
	n := lhs.Len()
	out := vector.NewBoolEmpty(n, nil)
	l := lhs.(*vector.Float)
	r := lhs.(*vector.Float)
	switch {
	case lx == nil && rx == nil:
		switch op {
		case "==":
			for k := uint32(0); k < n; k++ {
				if l.Values[k] == r.Values[k] {
					out.Set(k)
				}
			}
		case "!=":
			for k := uint32(0); k < n; k++ {
				if l.Values[k] != r.Values[k] {
					out.Set(k)
				}
			}
		case "<":
			for k := uint32(0); k < n; k++ {
				if l.Values[k] < r.Values[k] {
					out.Set(k)
				}
			}
		case "<=":
			for k := uint32(0); k < n; k++ {
				if l.Values[k] <= r.Values[k] {
					out.Set(k)
				}
			}
		default:
			panic(fmt.Sprintf("unknown op %q", op))
		}
	case lx != nil && rx == nil:
		switch op {
		case "==":
			for k := uint32(0); k < n; k++ {
				if l.Values[lx[k]] == r.Values[k] {
					out.Set(k)
				}
			}
		case "!=":
			for k := uint32(0); k < n; k++ {
				if l.Values[lx[k]] != r.Values[k] {
					out.Set(k)
				}
			}
		case "<":
			for k := uint32(0); k < n; k++ {
				if l.Values[lx[k]] < r.Values[k] {
					out.Set(k)
				}
			}
		case "<=":
			for k := uint32(0); k < n; k++ {
				if l.Values[lx[k]] <= r.Values[k] {
					out.Set(k)
				}
			}
		default:
			panic(fmt.Sprintf("unknown op %q", op))
		}
	case lx == nil && rx != nil:
		switch op {
		case "==":
			for k := uint32(0); k < n; k++ {
				if l.Values[k] == r.Values[rx[k]] {
					out.Set(k)
				}
			}
		case "!=":
			for k := uint32(0); k < n; k++ {
				if l.Values[k] != r.Values[rx[k]] {
					out.Set(k)
				}
			}
		case "<":
			for k := uint32(0); k < n; k++ {
				if l.Values[k] < r.Values[rx[k]] {
					out.Set(k)
				}
			}
		case "<=":
			for k := uint32(0); k < n; k++ {
				if l.Values[k] <= r.Values[rx[k]] {
					out.Set(k)
				}
			}
		default:
			panic(fmt.Sprintf("unknown op %q", op))
		}
	case lx != nil && rx != nil:
		switch op {
		case "==":
			for k := uint32(0); k < n; k++ {
				if l.Values[lx[k]] == r.Values[rx[k]] {
					out.Set(k)
				}
			}
		case "!=":
			for k := uint32(0); k < n; k++ {
				if l.Values[lx[k]] != r.Values[rx[k]] {
					out.Set(k)
				}
			}
		case "<":
			for k := uint32(0); k < n; k++ {
				if l.Values[lx[k]] < r.Values[rx[k]] {
					out.Set(k)
				}
			}
		case "<=":
			for k := uint32(0); k < n; k++ {
				if l.Values[lx[k]] <= r.Values[rx[k]] {
					out.Set(k)
				}
			}
		default:
			panic(fmt.Sprintf("unknown op %q", op))
		}
	}
	return out
}

func compareFloatConst(op string, lhs *vector.Float, idx []uint32, rhs *vector.Const) vector.Any {
	n := lhs.Len()
	out := vector.NewBoolEmpty(n, nil)
	literal, ok := rhs.AsFloat()
	if !ok {
		//XXX
		return nil
	}
	if idx == nil {
		switch op {
		case "==":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[k] == literal {
					out.Set(k)
				}
			}
		case "!=":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[k] != literal {
					out.Set(k)
				}
			}
		case "<":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[k] < literal {
					out.Set(k)
				}
			}
		case "<=":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[k] <= literal {
					out.Set(k)
				}
			}
		case ">":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[k] > literal {
					out.Set(k)
				}
			}
		case ">=":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[k] >= literal {
					out.Set(k)
				}
			}
		default:
			panic(fmt.Sprintf("unknown op %q", op))
		}
	} else {
		switch op {
		case "==":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[idx[k]] == literal {
					out.Set(k)
				}
			}
		case "!=":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[idx[k]] != literal {
					out.Set(k)
				}
			}
		case "<":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[idx[k]] < literal {
					out.Set(k)
				}
			}
		case "<=":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[idx[k]] <= literal {
					out.Set(k)
				}
			}
		case ">":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[idx[k]] > literal {
					out.Set(k)
				}
			}
		case ">=":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[idx[k]] >= literal {
					out.Set(k)
				}
			}
		}
	}
	return out
}

func compareFloatConsts(op string, lhs *vector.Const, rhs *vector.Const) vector.Any {
	l, _ := lhs.AsFloat()
	r, _ := rhs.AsFloat()
	var result bool
	switch op {
	case "==":
		result = l == r
	case "!=":
		result = l != r
	case "<":
		result = l < r
	case "<=":
		result = l <= r
	case ">":
		result = l > r
	case ">=":
		result = l >= r
	default:
		panic(fmt.Sprintf("unknown op %q", op))
	}
	return vector.NewConst(zed.NewBool(result), lhs.Len(), nil)
}

func compareInts(op string, lhs, rhs vector.Any, lx, rx []uint32) vector.Any {
	if lc, ok := lhs.(*vector.Const); ok {
		if rc, ok := rhs.(*vector.Const); ok {
			return compareIntConsts(op, lc, rc)
		}
		return compareIntConst(swapOp(op), rhs.(*vector.Int), rx, lc)
	}
	if rc, ok := rhs.(*vector.Const); ok {
		return compareIntConst(op, lhs.(*vector.Int), lx, rc)
	}
	n := lhs.Len()
	out := vector.NewBoolEmpty(n, nil)
	l := lhs.(*vector.Int)
	r := lhs.(*vector.Int)
	switch {
	case lx == nil && rx == nil:
		switch op {
		case "==":
			for k := uint32(0); k < n; k++ {
				if l.Values[k] == r.Values[k] {
					out.Set(k)
				}
			}
		case "!=":
			for k := uint32(0); k < n; k++ {
				if l.Values[k] != r.Values[k] {
					out.Set(k)
				}
			}
		case "<":
			for k := uint32(0); k < n; k++ {
				if l.Values[k] < r.Values[k] {
					out.Set(k)
				}
			}
		case "<=":
			for k := uint32(0); k < n; k++ {
				if l.Values[k] <= r.Values[k] {
					out.Set(k)
				}
			}
		default:
			panic(fmt.Sprintf("unknown op %q", op))
		}
	case lx != nil && rx == nil:
		switch op {
		case "==":
			for k := uint32(0); k < n; k++ {
				if l.Values[lx[k]] == r.Values[k] {
					out.Set(k)
				}
			}
		case "!=":
			for k := uint32(0); k < n; k++ {
				if l.Values[lx[k]] != r.Values[k] {
					out.Set(k)
				}
			}
		case "<":
			for k := uint32(0); k < n; k++ {
				if l.Values[lx[k]] < r.Values[k] {
					out.Set(k)
				}
			}
		case "<=":
			for k := uint32(0); k < n; k++ {
				if l.Values[lx[k]] <= r.Values[k] {
					out.Set(k)
				}
			}
		default:
			panic(fmt.Sprintf("unknown op %q", op))
		}
	case lx == nil && rx != nil:
		switch op {
		case "==":
			for k := uint32(0); k < n; k++ {
				if l.Values[k] == r.Values[rx[k]] {
					out.Set(k)
				}
			}
		case "!=":
			for k := uint32(0); k < n; k++ {
				if l.Values[k] != r.Values[rx[k]] {
					out.Set(k)
				}
			}
		case "<":
			for k := uint32(0); k < n; k++ {
				if l.Values[k] < r.Values[rx[k]] {
					out.Set(k)
				}
			}
		case "<=":
			for k := uint32(0); k < n; k++ {
				if l.Values[k] <= r.Values[rx[k]] {
					out.Set(k)
				}
			}
		default:
			panic(fmt.Sprintf("unknown op %q", op))
		}
	case lx != nil && rx != nil:
		switch op {
		case "==":
			for k := uint32(0); k < n; k++ {
				if l.Values[lx[k]] == r.Values[rx[k]] {
					out.Set(k)
				}
			}
		case "!=":
			for k := uint32(0); k < n; k++ {
				if l.Values[lx[k]] != r.Values[rx[k]] {
					out.Set(k)
				}
			}
		case "<":
			for k := uint32(0); k < n; k++ {
				if l.Values[lx[k]] < r.Values[rx[k]] {
					out.Set(k)
				}
			}
		case "<=":
			for k := uint32(0); k < n; k++ {
				if l.Values[lx[k]] <= r.Values[rx[k]] {
					out.Set(k)
				}
			}
		default:
			panic(fmt.Sprintf("unknown op %q", op))
		}
	}
	return out
}

func compareIntConst(op string, lhs *vector.Int, idx []uint32, rhs *vector.Const) vector.Any {
	n := lhs.Len()
	out := vector.NewBoolEmpty(n, nil)
	literal, ok := rhs.AsInt()
	if !ok {
		//XXX
		return nil
	}
	if idx == nil {
		switch op {
		case "==":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[k] == literal {
					out.Set(k)
				}
			}
		case "!=":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[k] != literal {
					out.Set(k)
				}
			}
		case "<":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[k] < literal {
					out.Set(k)
				}
			}
		case "<=":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[k] <= literal {
					out.Set(k)
				}
			}
		case ">":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[k] > literal {
					out.Set(k)
				}
			}
		case ">=":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[k] >= literal {
					out.Set(k)
				}
			}
		default:
			panic(fmt.Sprintf("unknown op %q", op))
		}
	} else {
		switch op {
		case "==":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[idx[k]] == literal {
					out.Set(k)
				}
			}
		case "!=":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[idx[k]] != literal {
					out.Set(k)
				}
			}
		case "<":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[idx[k]] < literal {
					out.Set(k)
				}
			}
		case "<=":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[idx[k]] <= literal {
					out.Set(k)
				}
			}
		case ">":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[idx[k]] > literal {
					out.Set(k)
				}
			}
		case ">=":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[idx[k]] >= literal {
					out.Set(k)
				}
			}
		}
	}
	return out
}

func compareIntConsts(op string, lhs *vector.Const, rhs *vector.Const) vector.Any {
	l, _ := lhs.AsInt()
	r, _ := rhs.AsInt()
	var result bool
	switch op {
	case "==":
		result = l == r
	case "!=":
		result = l != r
	case "<":
		result = l < r
	case "<=":
		result = l <= r
	case ">":
		result = l > r
	case ">=":
		result = l >= r
	default:
		panic(fmt.Sprintf("unknown op %q", op))
	}
	return vector.NewConst(zed.NewBool(result), lhs.Len(), nil)
}

func compareUints(op string, lhs, rhs vector.Any, lx, rx []uint32) vector.Any {
	if lc, ok := lhs.(*vector.Const); ok {
		if rc, ok := rhs.(*vector.Const); ok {
			return compareUintConsts(op, lc, rc)
		}
		return compareUintConst(swapOp(op), rhs.(*vector.Uint), rx, lc)
	}
	if rc, ok := rhs.(*vector.Const); ok {
		return compareUintConst(op, lhs.(*vector.Uint), lx, rc)
	}
	n := lhs.Len()
	out := vector.NewBoolEmpty(n, nil)
	l := lhs.(*vector.Uint)
	r := lhs.(*vector.Uint)
	switch {
	case lx == nil && rx == nil:
		switch op {
		case "==":
			for k := uint32(0); k < n; k++ {
				if l.Values[k] == r.Values[k] {
					out.Set(k)
				}
			}
		case "!=":
			for k := uint32(0); k < n; k++ {
				if l.Values[k] != r.Values[k] {
					out.Set(k)
				}
			}
		case "<":
			for k := uint32(0); k < n; k++ {
				if l.Values[k] < r.Values[k] {
					out.Set(k)
				}
			}
		case "<=":
			for k := uint32(0); k < n; k++ {
				if l.Values[k] <= r.Values[k] {
					out.Set(k)
				}
			}
		default:
			panic(fmt.Sprintf("unknown op %q", op))
		}
	case lx != nil && rx == nil:
		switch op {
		case "==":
			for k := uint32(0); k < n; k++ {
				if l.Values[lx[k]] == r.Values[k] {
					out.Set(k)
				}
			}
		case "!=":
			for k := uint32(0); k < n; k++ {
				if l.Values[lx[k]] != r.Values[k] {
					out.Set(k)
				}
			}
		case "<":
			for k := uint32(0); k < n; k++ {
				if l.Values[lx[k]] < r.Values[k] {
					out.Set(k)
				}
			}
		case "<=":
			for k := uint32(0); k < n; k++ {
				if l.Values[lx[k]] <= r.Values[k] {
					out.Set(k)
				}
			}
		default:
			panic(fmt.Sprintf("unknown op %q", op))
		}
	case lx == nil && rx != nil:
		switch op {
		case "==":
			for k := uint32(0); k < n; k++ {
				if l.Values[k] == r.Values[rx[k]] {
					out.Set(k)
				}
			}
		case "!=":
			for k := uint32(0); k < n; k++ {
				if l.Values[k] != r.Values[rx[k]] {
					out.Set(k)
				}
			}
		case "<":
			for k := uint32(0); k < n; k++ {
				if l.Values[k] < r.Values[rx[k]] {
					out.Set(k)
				}
			}
		case "<=":
			for k := uint32(0); k < n; k++ {
				if l.Values[k] <= r.Values[rx[k]] {
					out.Set(k)
				}
			}
		default:
			panic(fmt.Sprintf("unknown op %q", op))
		}
	case lx != nil && rx != nil:
		switch op {
		case "==":
			for k := uint32(0); k < n; k++ {
				if l.Values[lx[k]] == r.Values[rx[k]] {
					out.Set(k)
				}
			}
		case "!=":
			for k := uint32(0); k < n; k++ {
				if l.Values[lx[k]] != r.Values[rx[k]] {
					out.Set(k)
				}
			}
		case "<":
			for k := uint32(0); k < n; k++ {
				if l.Values[lx[k]] < r.Values[rx[k]] {
					out.Set(k)
				}
			}
		case "<=":
			for k := uint32(0); k < n; k++ {
				if l.Values[lx[k]] <= r.Values[rx[k]] {
					out.Set(k)
				}
			}
		default:
			panic(fmt.Sprintf("unknown op %q", op))
		}
	}
	return out
}

func compareUintConst(op string, lhs *vector.Uint, idx []uint32, rhs *vector.Const) vector.Any {
	n := lhs.Len()
	out := vector.NewBoolEmpty(n, nil)
	literal, ok := rhs.AsUint()
	if !ok {
		//XXX
		return nil
	}
	if idx == nil {
		switch op {
		case "==":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[k] == literal {
					out.Set(k)
				}
			}
		case "!=":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[k] != literal {
					out.Set(k)
				}
			}
		case "<":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[k] < literal {
					out.Set(k)
				}
			}
		case "<=":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[k] <= literal {
					out.Set(k)
				}
			}
		case ">":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[k] > literal {
					out.Set(k)
				}
			}
		case ">=":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[k] >= literal {
					out.Set(k)
				}
			}
		default:
			panic(fmt.Sprintf("unknown op %q", op))
		}
	} else {
		switch op {
		case "==":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[idx[k]] == literal {
					out.Set(k)
				}
			}
		case "!=":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[idx[k]] != literal {
					out.Set(k)
				}
			}
		case "<":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[idx[k]] < literal {
					out.Set(k)
				}
			}
		case "<=":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[idx[k]] <= literal {
					out.Set(k)
				}
			}
		case ">":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[idx[k]] > literal {
					out.Set(k)
				}
			}
		case ">=":
			for k := uint32(0); k < n; k++ {
				if lhs.Values[idx[k]] >= literal {
					out.Set(k)
				}
			}
		}
	}
	return out
}

func compareUintConsts(op string, lhs *vector.Const, rhs *vector.Const) vector.Any {
	l, _ := lhs.AsUint()
	r, _ := rhs.AsUint()
	var result bool
	switch op {
	case "==":
		result = l == r
	case "!=":
		result = l != r
	case "<":
		result = l < r
	case "<=":
		result = l <= r
	case ">":
		result = l > r
	case ">=":
		result = l >= r
	default:
		panic(fmt.Sprintf("unknown op %q", op))
	}
	return vector.NewConst(zed.NewBool(result), lhs.Len(), nil)
}

package expr

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/sam/expr/coerce"
	"github.com/brimdata/zed/vector"
)

// XXX work out stitch pattern... Eval can return wide variants and
// stitch ensures that the variant always stats at top.
// Define precisely: when multiple variants are input to an operator
// (like binary expr), we build aligned vectors from the variants
// not as a cross-product but only for vector pairs that overlap
// by slot.  We then run the operation on each vector pair then
// stich the result back into a wide variant.  Errors are always
// kept as the last element of the wide variant and may in turn be
// a variant themselves when error types diverge (e.g., because of stacking).

type Compare struct {
	zctx *zed.Context
	op   string
	lhs  Evaluator
	rhs  Evaluator
}

func NewCompare(zctx *zed.Context, lhs, rhs Evaluator, operator string) *Compare {
	return &Compare{zctx, operator, lhs, rhs}
}

//XXX break out const

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
	//XXX need to handle overflow (see sam)
	//XXX unions and variants
	//XXX nulls we can clean up nulls after the fact for primitive
	// types... unions/variants more complicated (variant/err) too
	// XXX need to support other primitives like strings, bytes, types, etc
	id := lhs.Type().ID()
	switch {
	case zed.IsFloat(id):
		return compareFloats(op, lhs, rhs)
	case zed.IsSigned(id):
		return compareInts(op, lhs, rhs)
	case zed.IsUnsigned(id):
		return compareUints(op, lhs, rhs)
	default:
		//XXX incompatible types
		return vector.NewStringError(c.zctx, coerce.IncompatibleTypes.Error(), lhs.Len())
	}
}

func compareFloats(op string, lhs, rhs vector.Any) vector.Any {
	n := lhs.Len()
	out := vector.NewBoolEmpty(n, nil)
	switch lhs := lhs.(type) {
	case *vector.Float:
		switch rhs := rhs.(type) {
		case *vector.Float:
			switch op {
			case "==":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[k] == rhs.Values[k] {
						out.Set(k)
					}
				}
			case "!=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[k] != rhs.Values[k] {
						out.Set(k)
					}
				}
			case "<":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[k] < rhs.Values[k] {
						out.Set(k)
					}
				}
			case "<=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[k] <= rhs.Values[k] {
						out.Set(k)
					}
				}
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		case *vector.DictFloat:
			switch op {
			case "==":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[k] == rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "!=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[k] != rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "<":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[k] < rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "<=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[k] <= rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		case *vector.Const:
			literal, ok := rhs.AsFloat()
			if !ok {
				//XXX
				return nil
			}
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
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		default:
			panic(fmt.Sprintf("bad type %T", rhs))
		}
	case *vector.DictFloat:
		switch rhs := rhs.(type) {
		case *vector.Float:
			switch op {
			case "==":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] == rhs.Values[k] {
						out.Set(k)
					}
				}
			case "!=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] != rhs.Values[k] {
						out.Set(k)
					}
				}
			case "<":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] < rhs.Values[k] {
						out.Set(k)
					}
				}
			case "<=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] <= rhs.Values[k] {
						out.Set(k)
					}
				}
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		case *vector.DictFloat:
			switch op {
			case "==":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] == rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "!=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] != rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "<":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] < rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "<=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] <= rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		case *vector.Const:
			literal, ok := rhs.AsFloat()
			if !ok {
				return nil
			}
			switch op {
			case "==":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] == literal {
						out.Set(k)
					}
				}
			case "!=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] != literal {
						out.Set(k)
					}
				}
			case "<":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] < literal {
						out.Set(k)
					}
				}
			case "<=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] <= literal {
						out.Set(k)
					}
				}
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		default:
			panic(fmt.Sprintf("bad type %T", rhs))
		}
	case *vector.Const:
		literal, ok := lhs.AsFloat()
		if !ok {
			return nil
		}
		switch rhs := rhs.(type) {
		case *vector.Float:
			switch op {
			case "==":
				for k := uint32(0); k < n; k++ {
					if literal == rhs.Values[k] {
						out.Set(k)
					}
				}
			case "!=":
				for k := uint32(0); k < n; k++ {
					if literal != rhs.Values[k] {
						out.Set(k)
					}
				}
			case "<":
				for k := uint32(0); k < n; k++ {
					if literal < rhs.Values[k] {
						out.Set(k)
					}
				}
			case "<=":
				for k := uint32(0); k < n; k++ {
					if literal <= rhs.Values[k] {
						out.Set(k)
					}
				}
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		case *vector.DictFloat:
			switch op {
			case "==":
				for k := uint32(0); k < n; k++ {
					if literal == rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "!=":
				for k := uint32(0); k < n; k++ {
					if literal != rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "<":
				for k := uint32(0); k < n; k++ {
					if literal < rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "<=":
				for k := uint32(0); k < n; k++ {
					if literal <= rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		case *vector.Const:
			left := literal
			right, ok := rhs.AsFloat()
			if !ok {
				//XXX
				return nil
			}
			switch op {
			case "==":
				if left == right {
					for k := uint32(0); k < n; k++ {
						out.Set(k)
					}
				}
			case "!=":
				if left != right {
					for k := uint32(0); k < n; k++ {
						out.Set(k)
					}
				}
			case "<":
				if left < right {
					for k := uint32(0); k < n; k++ {
						out.Set(k)
					}
				}
			case "<=":
				if left <= right {
					for k := uint32(0); k < n; k++ {
						out.Set(k)
					}
				}
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		default:
			panic(fmt.Sprintf("bad type %T", rhs))
		}
	default:
		panic(fmt.Sprintf("bad type %T", lhs))
	}
	return out
}

func compareInts(op string, lhs, rhs vector.Any) vector.Any {
	n := lhs.Len()
	out := vector.NewBoolEmpty(n, nil)
	switch lhs := lhs.(type) {
	case *vector.Int:
		switch rhs := rhs.(type) {
		case *vector.Int:
			switch op {
			case "==":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[k] == rhs.Values[k] {
						out.Set(k)
					}
				}
			case "!=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[k] != rhs.Values[k] {
						out.Set(k)
					}
				}
			case "<":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[k] < rhs.Values[k] {
						out.Set(k)
					}
				}
			case "<=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[k] <= rhs.Values[k] {
						out.Set(k)
					}
				}
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		case *vector.DictInt:
			switch op {
			case "==":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[k] == rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "!=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[k] != rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "<":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[k] < rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "<=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[k] <= rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		case *vector.Const:
			literal, ok := rhs.AsInt()
			if !ok {
				return nil
			}
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
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		default:
			panic(fmt.Sprintf("bad type %T", rhs))
		}
	case *vector.DictInt:
		switch rhs := rhs.(type) {
		case *vector.Int:
			switch op {
			case "==":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] == rhs.Values[k] {
						out.Set(k)
					}
				}
			case "!=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] != rhs.Values[k] {
						out.Set(k)
					}
				}
			case "<":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] < rhs.Values[k] {
						out.Set(k)
					}
				}
			case "<=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] <= rhs.Values[k] {
						out.Set(k)
					}
				}
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		case *vector.DictInt:
			switch op {
			case "==":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] == rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "!=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] != rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "<":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] < rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "<=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] <= rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		case *vector.Const:
			literal, ok := rhs.AsInt()
			if !ok {
				return nil
			}
			switch op {
			case "==":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] == literal {
						out.Set(k)
					}
				}
			case "!=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] != literal {
						out.Set(k)
					}
				}
			case "<":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] < literal {
						out.Set(k)
					}
				}
			case "<=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] <= literal {
						out.Set(k)
					}
				}
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		default:
			panic(fmt.Sprintf("bad type %T", rhs))
		}
	case *vector.Const:
		literal, ok := lhs.AsInt()
		if !ok {
			return nil
		}
		switch rhs := rhs.(type) {
		case *vector.Int:
			switch op {
			case "==":
				for k := uint32(0); k < n; k++ {
					if literal == rhs.Values[k] {
						out.Set(k)
					}
				}
			case "!=":
				for k := uint32(0); k < n; k++ {
					if literal != rhs.Values[k] {
						out.Set(k)
					}
				}
			case "<":
				for k := uint32(0); k < n; k++ {
					if literal < rhs.Values[k] {
						out.Set(k)
					}
				}
			case "<=":
				for k := uint32(0); k < n; k++ {
					if literal <= rhs.Values[k] {
						out.Set(k)
					}
				}
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		case *vector.DictInt:
			switch op {
			case "==":
				for k := uint32(0); k < n; k++ {
					if literal == rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "!=":
				for k := uint32(0); k < n; k++ {
					if literal != rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "<":
				for k := uint32(0); k < n; k++ {
					if literal < rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "<=":
				for k := uint32(0); k < n; k++ {
					if literal <= rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		case *vector.Const:
			left := literal
			right, ok := rhs.AsInt()
			if !ok {
				return nil
			}
			switch op {
			case "==":
				if left == right {
					for k := uint32(0); k < n; k++ {
						out.Set(k)
					}
				}
			case "!=":
				if left != right {
					for k := uint32(0); k < n; k++ {
						out.Set(k)
					}
				}
			case "<":
				if left < right {
					for k := uint32(0); k < n; k++ {
						out.Set(k)
					}
				}
			case "<=":
				if left <= right {
					for k := uint32(0); k < n; k++ {
						out.Set(k)
					}
				}
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		default:
			panic(fmt.Sprintf("bad type %T", rhs))
		}
	default:
		panic(fmt.Sprintf("bad type %T", lhs))
	}
	return out
}

func compareUints(op string, lhs, rhs vector.Any) vector.Any {
	n := lhs.Len()
	out := vector.NewBoolEmpty(n, nil)
	switch lhs := lhs.(type) {
	case *vector.Uint:
		switch rhs := rhs.(type) {
		case *vector.Uint:
			switch op {
			case "==":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[k] == rhs.Values[k] {
						out.Set(k)
					}
				}
			case "!=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[k] != rhs.Values[k] {
						out.Set(k)
					}
				}
			case "<":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[k] < rhs.Values[k] {
						out.Set(k)
					}
				}
			case "<=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[k] <= rhs.Values[k] {
						out.Set(k)
					}
				}
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		case *vector.DictUint:
			switch op {
			case "==":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[k] == rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "!=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[k] != rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "<":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[k] < rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "<=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[k] <= rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		case *vector.Const:
			literal, ok := rhs.AsUint()
			if !ok {
				return nil
			}
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
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		default:
			panic(fmt.Sprintf("bad type %T", rhs))
		}
	case *vector.DictUint:
		switch rhs := rhs.(type) {
		case *vector.Uint:
			switch op {
			case "==":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] == rhs.Values[k] {
						out.Set(k)
					}
				}
			case "!=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] != rhs.Values[k] {
						out.Set(k)
					}
				}
			case "<":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] < rhs.Values[k] {
						out.Set(k)
					}
				}
			case "<=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] <= rhs.Values[k] {
						out.Set(k)
					}
				}
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		case *vector.DictUint:
			switch op {
			case "==":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] == rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "!=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] != rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "<":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] < rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "<=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] <= rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			}
		case *vector.Const:
			literal, ok := rhs.AsUint()
			if !ok {
				return nil
			}
			switch op {
			case "==":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] == literal {
						out.Set(k)
					}
				}
			case "!=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] != literal {
						out.Set(k)
					}
				}
			case "<":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] < literal {
						out.Set(k)
					}
				}
			case "<=":
				for k := uint32(0); k < n; k++ {
					if lhs.Values[lhs.Tags[k]] <= literal {
						out.Set(k)
					}
				}
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		default:
			panic(fmt.Sprintf("bad type %T", rhs))
		}
	case *vector.Const:
		literal, ok := lhs.AsUint()
		if !ok {
			return nil
		}
		switch rhs := rhs.(type) {
		case *vector.Uint:
			switch op {
			case "==":
				for k := uint32(0); k < n; k++ {
					if literal == rhs.Values[k] {
						out.Set(k)
					}
				}
			case "!=":
				for k := uint32(0); k < n; k++ {
					if literal != rhs.Values[k] {
						out.Set(k)
					}
				}
			case "<":
				for k := uint32(0); k < n; k++ {
					if literal < rhs.Values[k] {
						out.Set(k)
					}
				}
			case "<=":
				for k := uint32(0); k < n; k++ {
					if literal <= rhs.Values[k] {
						out.Set(k)
					}
				}
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		case *vector.DictUint:
			switch op {
			case "==":
				for k := uint32(0); k < n; k++ {
					if literal == rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "!=":
				for k := uint32(0); k < n; k++ {
					if literal != rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "<":
				for k := uint32(0); k < n; k++ {
					if literal < rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			case "<=":
				for k := uint32(0); k < n; k++ {
					if literal <= rhs.Values[rhs.Tags[k]] {
						out.Set(k)
					}
				}
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		case *vector.Const:
			left := literal
			right, ok := rhs.AsUint()
			if !ok {
				return nil
			}
			switch op {
			case "==":
				if left == right {
					for k := uint32(0); k < n; k++ {
						out.Set(k)
					}
				}
			case "!=":
				if left != right {
					for k := uint32(0); k < n; k++ {
						out.Set(k)
					}
				}
			case "<":
				if left < right {
					for k := uint32(0); k < n; k++ {
						out.Set(k)
					}
				}
			case "<=":
				if left <= right {
					for k := uint32(0); k < n; k++ {
						out.Set(k)
					}
				}
			default:
				panic(fmt.Sprintf("unknown op %q", op))
			}
		default:
			panic(fmt.Sprintf("bad type %T", rhs))
		}
	default:
		panic(fmt.Sprintf("bad type %T", lhs))
	}
	return out
}

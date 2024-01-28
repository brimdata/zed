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
	lhs := vector.Under(c.lhs.Eval(val))
	rhs := vector.Under(c.rhs.Eval(val))
	left, right, _ := coerceVals(c.zctx, lhs, rhs)
	op := c.op
	switch op {
	case ">=":
		op = "<"
		left, right = right, left
	case ">":
		op = "<="
		left, right = right, left
	}
	//XXX need to handle overflow (see sam)
	//XXX unions and variants
	//XXX nulls we can clean up nulls after the fact for primitive
	// types... unions/variants more complicated (variant/err) too
	switch lhs := lhs.(type) {
	case *vector.Float, *vector.DictFloat:
		return compareFloats(op, left, right)
	case *vector.Int:
		return compareInts(op, left, right)
	case *vector.Uint:
		return compareUints(op, left, right)
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
			out := vector.NewBoolEmpty(lhs.Len(), nil)
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
			out := vector.NewBoolEmpty(lhs.Len(), nil)
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
			out := vector.NewBoolEmpty(lhs.Len(), nil)
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

package expr

import (
	"github.com/brimdata/zed"
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

func (c *Compare) Eval(val vector.Any) vector.Any {
	lhs := c.lhs.Eval(val)
	rhs := c.rhs.Eval(val)
	clhs, crhs, ok := coerce(lhs, rhs)
	//XXX need to handle overflow (see sam)
	switch lhs := lhs.(type) {
	case *vector.Float:
		return compareFloats(c.op, lhs, rhs.(*vector.Float))
	case *vector.Int:
		return compareInts(c.op, lhs, rhs.(*vector.Float))
	case *vector.Uint:
		return compareUints(c.op, lhs, rhs.(*vector.Float))
	default:
		//XXX incompatible types
		return vector.NewError()
	}
}

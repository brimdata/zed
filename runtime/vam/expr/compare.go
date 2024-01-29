package expr

import (
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

	if val := cmpLookup(op, lhs, rhs)(lhs, rhs); val != nil {
		return val
	}
	return vector.NewStringError(c.zctx, coerce.IncompatibleTypes.Error(), lhs.Len())
}

const (
	// type
	cFloat = 0
	cInt   = 1
	cUint  = 2
	// kind
	cFlat  = 0
	cDict  = 1
	cConst = 2
	// op
	cEq = 0
	cNe = 1
	cLt = 2
	cLe = 3
)

func enc(op, lkind, ltype, rkind, rtype int) int {
	return ltype<<8 | lkind<<6 | rtype<<4 | rkind<<2 | op
}

func init() {
	cmpLUT = make(map[int]compareFn)

	cmpLUT[enc(cEq, cFlat, cFloat, cFlat, cFloat)] = eq_FlatFloat_FlatFloat
	cmpLUT[enc(cNe, cFlat, cFloat, cFlat, cFloat)] = ne_FlatFloat_FlatFloat
	cmpLUT[enc(cLt, cFlat, cFloat, cFlat, cFloat)] = lt_FlatFloat_FlatFloat
	cmpLUT[enc(cLe, cFlat, cFloat, cFlat, cFloat)] = le_FlatFloat_FlatFloat

	cmpLUT[enc(cEq, cFlat, cFloat, cDict, cFloat)] = eq_FlatFloat_DictFloat
	cmpLUT[enc(cNe, cFlat, cFloat, cDict, cFloat)] = ne_FlatFloat_DictFloat
	cmpLUT[enc(cLt, cFlat, cFloat, cDict, cFloat)] = lt_FlatFloat_DictFloat
	cmpLUT[enc(cLe, cFlat, cFloat, cDict, cFloat)] = le_FlatFloat_DictFloat

	cmpLUT[enc(cEq, cFlat, cFloat, cConst, cFloat)] = eq_FlatFloat_ConstFloat
	cmpLUT[enc(cNe, cFlat, cFloat, cConst, cFloat)] = ne_FlatFloat_ConstFloat
	cmpLUT[enc(cLt, cFlat, cFloat, cConst, cFloat)] = lt_FlatFloat_ConstFloat
	cmpLUT[enc(cLe, cFlat, cFloat, cConst, cFloat)] = le_FlatFloat_ConstFloat

	cmpLUT[enc(cEq, cFlat, cDict, cFlat, cFloat)] = eq_DictFloat_FlatFloat
	cmpLUT[enc(cNe, cFlat, cDict, cFlat, cFloat)] = ne_DictFloat_FlatFloat
	cmpLUT[enc(cLt, cFlat, cDict, cFlat, cFloat)] = lt_DictFloat_FlatFloat
	cmpLUT[enc(cLe, cFlat, cDict, cFlat, cFloat)] = le_DictFloat_FlatFloat

	cmpLUT[enc(cEq, cFlat, cDict, cDict, cFloat)] = eq_DictFloat_DictFloat
	cmpLUT[enc(cNe, cFlat, cDict, cDict, cFloat)] = ne_DictFloat_DictFloat
	cmpLUT[enc(cLt, cFlat, cDict, cDict, cFloat)] = lt_DictFloat_DictFloat
	cmpLUT[enc(cLe, cFlat, cDict, cDict, cFloat)] = le_DictFloat_DictFloat

	cmpLUT[enc(cEq, cFlat, cDict, cConst, cFloat)] = eq_DictFloat_ConstFloat
	cmpLUT[enc(cNe, cFlat, cDict, cConst, cFloat)] = ne_DictFloat_ConstFloat
	cmpLUT[enc(cLt, cFlat, cDict, cConst, cFloat)] = lt_DictFloat_ConstFloat
	cmpLUT[enc(cLe, cFlat, cDict, cConst, cFloat)] = le_DictFloat_ConstFloat

	cmpLUT[enc(cEq, cFlat, cDict, cFlat, cFloat)] = eq_ConstFloat_FlatFloat
	cmpLUT[enc(cNe, cFlat, cDict, cFlat, cFloat)] = ne_ConstFloat_FlatFloat
	cmpLUT[enc(cLt, cFlat, cDict, cFlat, cFloat)] = lt_ConstFloat_FlatFloat
	cmpLUT[enc(cLe, cFlat, cDict, cFlat, cFloat)] = le_ConstFloat_FlatFloat

	cmpLUT[enc(cEq, cFlat, cDict, cDict, cFloat)] = eq_ConstFloat_DictFloat
	cmpLUT[enc(cNe, cFlat, cDict, cDict, cFloat)] = ne_ConstFloat_DictFloat
	cmpLUT[enc(cLt, cFlat, cDict, cDict, cFloat)] = lt_ConstFloat_DictFloat
	cmpLUT[enc(cLe, cFlat, cDict, cDict, cFloat)] = le_ConstFloat_DictFloat

	cmpLUT[enc(cEq, cFlat, cDict, cConst, cFloat)] = eq_ConstFloat_ConstFloat
	cmpLUT[enc(cNe, cFlat, cDict, cConst, cFloat)] = ne_ConstFloat_ConstFloat
	cmpLUT[enc(cLt, cFlat, cDict, cConst, cFloat)] = lt_ConstFloat_ConstFloat
	cmpLUT[enc(cLe, cFlat, cDict, cConst, cFloat)] = le_ConstFloat_ConstFloat
}

type compareFn func(vector.Any, vector.Any) vector.Any

var cmpLUT map[int]compareFn

func cmpLookup(op string, lhs, rhs vector.Any) compareFn {
	return nil //XXX
}

func eq_FlatFloat_FlatFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	lhs := l.(*vector.Float)
	rhs := l.(*vector.Float)
	for k := uint32(0); k < n; k++ {
		if lhs.Values[k] == rhs.Values[k] {
			out.Set(k)
		}
	}
	return out
}

func ne_FlatFloat_FlatFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	lhs := l.(*vector.Float)
	rhs := l.(*vector.Float)
	for k := uint32(0); k < n; k++ {
		if lhs.Values[k] != rhs.Values[k] {
			out.Set(k)
		}
	}
	return out
}

func lt_FlatFloat_FlatFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	lhs := l.(*vector.Float)
	rhs := l.(*vector.Float)
	for k := uint32(0); k < n; k++ {
		if lhs.Values[k] < rhs.Values[k] {
			out.Set(k)
		}
	}
	return out
}

func le_FlatFloat_FlatFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	lhs := l.(*vector.Float)
	rhs := l.(*vector.Float)
	for k := uint32(0); k < n; k++ {
		if lhs.Values[k] <= rhs.Values[k] {
			out.Set(k)
		}
	}
	return out
}

func eq_FlatFloat_DictFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	lhs := l.(*vector.Float)
	rhs := l.(*vector.DictFloat)
	for k := uint32(0); k < n; k++ {
		if lhs.Values[k] == rhs.Values[rhs.Tags[k]] {
			out.Set(k)
		}
	}
	return out
}

func ne_FlatFloat_DictFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	lhs := l.(*vector.Float)
	rhs := l.(*vector.DictFloat)
	for k := uint32(0); k < n; k++ {
		if lhs.Values[k] != rhs.Values[rhs.Tags[k]] {
			out.Set(k)
		}
	}
	return out
}

func lt_FlatFloat_DictFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	lhs := l.(*vector.Float)
	rhs := l.(*vector.DictFloat)
	for k := uint32(0); k < n; k++ {
		if lhs.Values[k] < rhs.Values[rhs.Tags[k]] {
			out.Set(k)
		}
	}
	return out
}

func le_FlatFloat_DictFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	lhs := l.(*vector.Float)
	rhs := l.(*vector.DictFloat)
	for k := uint32(0); k < n; k++ {
		if lhs.Values[k] <= rhs.Values[rhs.Tags[k]] {
			out.Set(k)
		}
	}
	return out
}

func eq_FlatFloat_ConstFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	lhs := l.(*vector.Float)
	rhs := l.(*vector.Const)
	literal, ok := rhs.AsFloat()
	if !ok {
		return nil
	}
	for k := uint32(0); k < n; k++ {
		if lhs.Values[k] == literal {
			out.Set(k)
		}
	}
	return out
}

func ne_FlatFloat_ConstFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	lhs := l.(*vector.Float)
	rhs := l.(*vector.Const)
	literal, ok := rhs.AsFloat()
	if !ok {
		return nil
	}
	for k := uint32(0); k < n; k++ {
		if lhs.Values[k] != literal {
			out.Set(k)
		}
	}
	return out
}

func lt_FlatFloat_ConstFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	lhs := l.(*vector.Float)
	rhs := l.(*vector.Const)
	literal, ok := rhs.AsFloat()
	if !ok {
		return nil
	}
	for k := uint32(0); k < n; k++ {
		if lhs.Values[k] < literal {
			out.Set(k)
		}
	}
	return out
}

func le_FlatFloat_ConstFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	lhs := l.(*vector.Float)
	rhs := l.(*vector.Const)
	literal, ok := rhs.AsFloat()
	if !ok {
		return nil
	}
	for k := uint32(0); k < n; k++ {
		if lhs.Values[k] <= literal {
			out.Set(k)
		}
	}
	return out
}

func eq_DictFloat_FlatFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	lhs := l.(*vector.DictFloat)
	rhs := l.(*vector.Float)
	for k := uint32(0); k < n; k++ {
		if lhs.Values[lhs.Tags[k]] == rhs.Values[k] {
			out.Set(k)
		}
	}
	return out
}

func ne_DictFloat_FlatFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	lhs := l.(*vector.DictFloat)
	rhs := l.(*vector.Float)
	for k := uint32(0); k < n; k++ {
		if lhs.Values[lhs.Tags[k]] != rhs.Values[k] {
			out.Set(k)
		}
	}
	return out
}

func lt_DictFloat_FlatFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	lhs := l.(*vector.DictFloat)
	rhs := l.(*vector.Float)
	for k := uint32(0); k < n; k++ {
		if lhs.Values[lhs.Tags[k]] < rhs.Values[k] {
			out.Set(k)
		}
	}
	return out
}

func le_DictFloat_FlatFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	lhs := l.(*vector.DictFloat)
	rhs := l.(*vector.Float)
	for k := uint32(0); k < n; k++ {
		if lhs.Values[lhs.Tags[k]] <= rhs.Values[k] {
			out.Set(k)
		}
	}
	return out
}

func eq_DictFloat_DictFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	lhs := l.(*vector.DictFloat)
	rhs := l.(*vector.DictFloat)
	for k := uint32(0); k < n; k++ {
		if lhs.Values[lhs.Tags[k]] == rhs.Values[rhs.Tags[k]] {
			out.Set(k)
		}
	}
	return out
}

func ne_DictFloat_DictFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	lhs := l.(*vector.DictFloat)
	rhs := l.(*vector.DictFloat)
	for k := uint32(0); k < n; k++ {
		if lhs.Values[lhs.Tags[k]] != rhs.Values[rhs.Tags[k]] {
			out.Set(k)
		}
	}
	return out
}

func lt_DictFloat_DictFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	lhs := l.(*vector.DictFloat)
	rhs := l.(*vector.DictFloat)
	for k := uint32(0); k < n; k++ {
		if lhs.Values[lhs.Tags[k]] < rhs.Values[rhs.Tags[k]] {
			out.Set(k)
		}
	}
	return out
}

func le_DictFloat_DictFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	lhs := l.(*vector.DictFloat)
	rhs := l.(*vector.DictFloat)
	for k := uint32(0); k < n; k++ {
		if lhs.Values[lhs.Tags[k]] <= rhs.Values[rhs.Tags[k]] {
			out.Set(k)
		}
	}
	return out
}

func eq_DictFloat_ConstFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	lhs := l.(*vector.DictFloat)
	rhs := l.(*vector.Const)
	literal, ok := rhs.AsFloat()
	if !ok {
		return nil
	}
	for k := uint32(0); k < n; k++ {
		if lhs.Values[lhs.Tags[k]] == literal {
			out.Set(k)
		}
	}
	return out
}

func ne_DictFloat_ConstFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	lhs := l.(*vector.DictFloat)
	rhs := l.(*vector.Const)
	literal, ok := rhs.AsFloat()
	if !ok {
		return nil
	}
	for k := uint32(0); k < n; k++ {
		if lhs.Values[lhs.Tags[k]] != literal {
			out.Set(k)
		}
	}
	return out
}

func lt_DictFloat_ConstFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	lhs := l.(*vector.DictFloat)
	rhs := l.(*vector.Const)
	literal, ok := rhs.AsFloat()
	if !ok {
		return nil
	}
	for k := uint32(0); k < n; k++ {
		if lhs.Values[lhs.Tags[k]] < literal {
			out.Set(k)
		}
	}
	return out
}

func le_DictFloat_ConstFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	lhs := l.(*vector.DictFloat)
	rhs := l.(*vector.Const)
	literal, ok := rhs.AsFloat()
	if !ok {
		return nil
	}
	for k := uint32(0); k < n; k++ {
		if lhs.Values[lhs.Tags[k]] <= literal {
			out.Set(k)
		}
	}
	return out
}

func eq_ConstFloat_FlatFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	literal, ok := l.(*vector.Const).AsFloat()
	if !ok {
		return nil
	}
	rhs := l.(*vector.Float)
	for k := uint32(0); k < n; k++ {
		if literal == rhs.Values[k] {
			out.Set(k)
		}
	}
	return out
}

func ne_ConstFloat_FlatFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	literal, ok := l.(*vector.Const).AsFloat()
	if !ok {
		return nil
	}
	rhs := l.(*vector.Float)
	for k := uint32(0); k < n; k++ {
		if literal != rhs.Values[k] {
			out.Set(k)
		}
	}
	return out
}

func lt_ConstFloat_FlatFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	literal, ok := l.(*vector.Const).AsFloat()
	if !ok {
		return nil
	}
	rhs := l.(*vector.Float)
	for k := uint32(0); k < n; k++ {
		if literal < rhs.Values[k] {
			out.Set(k)
		}
	}
	return out
}

func le_ConstFloat_FlatFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	literal, ok := l.(*vector.Const).AsFloat()
	if !ok {
		return nil
	}
	rhs := l.(*vector.Float)
	for k := uint32(0); k < n; k++ {
		if literal <= rhs.Values[k] {
			out.Set(k)
		}
	}
	return out
}

func eq_ConstFloat_DictFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	literal, ok := l.(*vector.Const).AsFloat()
	if !ok {
		return nil
	}
	rhs := l.(*vector.DictFloat)
	for k := uint32(0); k < n; k++ {
		if literal == rhs.Values[rhs.Tags[k]] {
			out.Set(k)
		}
	}
	return out
}

func ne_ConstFloat_DictFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	literal, ok := l.(*vector.Const).AsFloat()
	if !ok {
		return nil
	}
	rhs := l.(*vector.DictFloat)
	for k := uint32(0); k < n; k++ {
		if literal != rhs.Values[rhs.Tags[k]] {
			out.Set(k)
		}
	}
	return out
}

func lt_ConstFloat_DictFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	literal, ok := l.(*vector.Const).AsFloat()
	if !ok {
		return nil
	}
	rhs := l.(*vector.DictFloat)
	for k := uint32(0); k < n; k++ {
		if literal < rhs.Values[rhs.Tags[k]] {
			out.Set(k)
		}
	}
	return out
}

func le_ConstFloat_DictFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	literal, ok := l.(*vector.Const).AsFloat()
	if !ok {
		return nil
	}
	rhs := l.(*vector.DictFloat)
	for k := uint32(0); k < n; k++ {
		if literal <= rhs.Values[rhs.Tags[k]] {
			out.Set(k)
		}
	}
	return out
}

func eq_ConstFloat_ConstFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	left, ok := l.(*vector.Const).AsFloat()
	if !ok {
		return nil
	}
	right, ok := r.(*vector.Const).AsFloat()
	if !ok {
		return nil
	}
	if left == right {
		for k := uint32(0); k < n; k++ {
			out.Set(k)
		}
	}
	return out
}

func ne_ConstFloat_ConstFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	left, ok := l.(*vector.Const).AsFloat()
	if !ok {
		return nil
	}
	right, ok := r.(*vector.Const).AsFloat()
	if !ok {
		return nil
	}
	if left != right {
		for k := uint32(0); k < n; k++ {
			out.Set(k)
		}
	}
	return out
}

func lt_ConstFloat_ConstFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	left, ok := l.(*vector.Const).AsFloat()
	if !ok {
		return nil
	}
	right, ok := r.(*vector.Const).AsFloat()
	if !ok {
		return nil
	}
	if left < right {
		for k := uint32(0); k < n; k++ {
			out.Set(k)
		}
	}
	return out
}

func le_ConstFloat_ConstFloat(l, r vector.Any) vector.Any {
	n := uint32(l.Len())
	out := vector.NewBoolEmpty(n, nil)
	left, ok := l.(*vector.Const).AsFloat()
	if !ok {
		return nil
	}
	right, ok := r.(*vector.Const).AsFloat()
	if !ok {
		return nil
	}
	if left <= right {
		for k := uint32(0); k < n; k++ {
			out.Set(k)
		}
	}
	return out
}

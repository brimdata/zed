package op

import (
	"github.com/superdb/super"
	"github.com/superdb/super/runtime/expr"
	"github.com/superdb/super/vector"
	"github.com/superdb/super/vector/vio"
)

type Filter struct {
	sctx   *super.Context
	parent vio.Puller
	expr   expr.Evaluator
	catch  expr.Evaluator
}

// NewFilter returns a filter that outputs the values of parent for which expr
// is true.  If catch is not nil, it handles the errors expr produces: catch is
// evaluated on the vector of those errors, yielding a value for each, and
// those values are output in place of the erroring values, in input order.
// If catch is nil, values on which expr errs are dropped.
func NewFilter(sctx *super.Context, parent vio.Puller, expr, catch expr.Evaluator) *Filter {
	return &Filter{sctx, parent, expr, catch}
}

func (f *Filter) Pull(done bool) (vector.Any, error) {
	for {
		vec, err := f.parent.Pull(done)
		if vec == nil || err != nil {
			return nil, err
		}
		mask := f.expr.Eval(vec)
		if f.catch == nil {
			if masked, ok := applyMask(vec, mask); ok {
				return masked, nil
			}
			continue
		}
		if out := f.filterAndCatch(vec, mask); out != nil {
			return out, nil
		}
	}
}

// filterAndCatch returns the values of vec for which mask is true merged, in
// input order, with the results of the catch handler for each error in mask.
// It returns nil if there's nothing to output.
func (f *Filter) filterAndCatch(vec, mask vector.Any) vector.Any {
	keep, other := expr.BoolMask(mask)
	var errs vector.Any
	var errSlots []uint32
	if !other.IsEmpty() {
		errs, errSlots = expr.CatchSlots(f.sctx, expr.CatchError, mask, "where")
	}
	if len(errSlots) == 0 {
		masked, _ := applyMask(vec, mask)
		return masked
	}
	return mergeCaught(vec, keep.ToArray(), f.catch.Eval(errs), errSlots)
}

// mergeCaught returns the values of vec at the slots in keep merged, in slot
// order, with the values of vals, which belong at the slots in valSlots.
// keep and valSlots must be sorted and disjoint and vals must have a value
// for each slot in valSlots.
func mergeCaught(vec vector.Any, keep []uint32, vals vector.Any, valSlots []uint32) vector.Any {
	if len(keep) == 0 {
		return vals
	}
	tags := make([]uint32, 0, len(keep)+len(valSlots))
	i, j := 0, 0
	for i < len(keep) && j < len(valSlots) {
		if keep[i] < valSlots[j] {
			tags = append(tags, 0)
			i++
		} else {
			tags = append(tags, 1)
			j++
		}
	}
	for ; i < len(keep); i++ {
		tags = append(tags, 0)
	}
	for ; j < len(valSlots); j++ {
		tags = append(tags, 1)
	}
	return vector.NewDynamic(tags, []vector.Any{vector.Pick(vec, keep), vals})
}

// applyMask applies the mask vector mask to vec.  Elements of mask that are not
// Boolean are considered false.
func applyMask(vec, mask vector.Any) (vector.Any, bool) {
	// errors are ignored for filters
	b, _ := expr.BoolMask(mask)
	if b.IsEmpty() {
		return nil, false
	}
	if b.GetCardinality() == uint64(mask.Len()) {
		return vec, true
	}
	return vector.Pick(vec, b.ToArray()), true
}

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
}

func NewFilter(sctx *super.Context, parent vio.Puller, expr expr.Evaluator) *Filter {
	return &Filter{sctx, parent, expr}
}

func (f *Filter) Pull(done bool) (vector.Any, error) {
	for {
		vec, err := f.parent.Pull(done)
		if vec == nil || err != nil {
			return nil, err
		}
		if out := f.filter(vec); out != nil {
			return out, nil
		}
	}
}

// filter returns the values of vec for which f.expr is true, merged in input
// order with any errors f.expr produced, which are side errors that would
// otherwise vanish since an error isn't true.  It returns nil if there's
// nothing to output.
func (f *Filter) filter(vec vector.Any) vector.Any {
	mask := f.expr.Eval(vec)
	keep, other := expr.BoolMask(mask)
	var errs vector.Any
	var errSlots []uint32
	if !other.IsEmpty() {
		errs, errSlots = expr.CatchSlots(f.sctx, expr.CatchError|expr.CatchWrapError, mask, "where")
	}
	if keep.IsEmpty() && len(errSlots) == 0 {
		return nil
	}
	return withSideErrors(vec, keep.ToArray(), errs, errSlots)
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

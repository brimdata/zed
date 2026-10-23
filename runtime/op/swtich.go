package op

import (
	"context"

	"github.com/RoaringBitmap/roaring/v2"
	"github.com/superdb/super"
	"github.com/superdb/super/runtime/expr"
	"github.com/superdb/super/vector"
	"github.com/superdb/super/vector/vio"
)

type Switch struct {
	sctx   *super.Context
	router *router
	cases  []expr.Evaluator
}

func NewSwitch(ctx context.Context, sctx *super.Context, parent vio.Puller) *Switch {
	s := &Switch{sctx: sctx}
	s.router = newRouter(ctx, s, parent)
	return s
}

func (s *Switch) AddCase(e expr.Evaluator) vio.Puller {
	s.cases = append(s.cases, e)
	return s.router.addRoute()
}

func (s *Switch) forward(vec vector.Any) bool {
	doneMap := roaring.New()
	for i, c := range s.cases {
		maskVec := c.Eval(vec)
		boolMap, other := expr.BoolMask(maskVec)
		boolMap.AndNot(doneMap)
		var errs vector.Any
		var errSlots []uint32
		if !other.IsEmpty() {
			// A case's errors are side errors sent down its route.  An error
			// doesn't claim its value, so later cases still see it, but skip
			// values an earlier case already took.
			errs, errSlots = expr.CatchSlots(s.sctx, expr.CatchError|expr.CatchWrapError, maskVec, "switch")
			if len(errSlots) > 0 {
				errs, errSlots = notDone(errs, errSlots, doneMap)
			}
		}
		doneMap.Or(boolMap)
		if boolMap.IsEmpty() && len(errSlots) == 0 {
			continue
		}
		vec2 := withSideErrors(vec, boolMap.ToArray(), errs, errSlots)
		if !s.router.routes[i].send(vec2, nil) {
			return false
		}
	}
	return true
}

// notDone returns the values of errs and their slots, dropping those whose
// slots are in done.
func notDone(errs vector.Any, slots []uint32, done *roaring.Bitmap) (vector.Any, []uint32) {
	var index, keep []uint32
	for k, slot := range slots {
		if !done.Contains(slot) {
			index = append(index, uint32(k))
			keep = append(keep, slot)
		}
	}
	if len(index) == len(slots) {
		return errs, slots
	}
	return vector.Pick(errs, index), keep
}

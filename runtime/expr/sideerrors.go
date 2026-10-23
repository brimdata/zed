package expr

import (
	"github.com/RoaringBitmap/roaring/v2"
	"github.com/superdb/super"
	"github.com/superdb/super/vector"
)

// CatchSlots applies Catch with opts to each type of value in vec, after
// ripping fusions and unions so a mixed-type vector is examined type by type.
// It returns the caught values in slot order along with their slots in vec,
// or nil, nil if nothing is caught.  Operators use it to find side errors:
// errors produced while evaluating an expression whose result the operator
// consumes rather than outputs (e.g., a where predicate or a sort key).
func CatchSlots(sctx *super.Context, opts int, vec vector.Any, msg string) (vector.Any, []uint32) {
	vec = vector.Apply(vector.ApplyRipFusions|vector.ApplyRipUnions, func(vecs ...vector.Any) vector.Any {
		return vecs[0]
	}, vec)
	dynamic, ok := vec.(*vector.Dynamic)
	if !ok {
		if out, ok := Catch(sctx, opts, vec, msg); ok {
			return out, allSlots(vec.Len())
		}
		return nil, nil
	}
	// part[tag] is the index in parts of the caught values for tag, or -1.
	part := make([]int, len(dynamic.Values))
	var parts []vector.Any
	for tag, vec := range dynamic.Values {
		part[tag] = -1
		if out, ok := Catch(sctx, opts, vec, msg); ok {
			part[tag] = len(parts)
			parts = append(parts, out)
		}
	}
	if len(parts) == 0 {
		return nil, nil
	}
	var slots, tags []uint32
	for slot, tag := range dynamic.Tags {
		if p := part[tag]; p >= 0 {
			slots = append(slots, uint32(slot))
			tags = append(tags, uint32(p))
		}
	}
	if len(parts) == 1 {
		return parts[0], slots
	}
	return vector.NewDynamic(tags, parts), slots
}

// PushdownMask returns the slots of a pushed-down filter's result that a scan
// must keep: those that are true and those that are errors.  Error slots are
// kept so the filter operator downstream can report them as side errors
// rather than the scan silently dropping them.
func PushdownMask(sctx *super.Context, result vector.Any) *roaring.Bitmap {
	keep, other := BoolMask(result)
	if other.IsEmpty() {
		return keep
	}
	if _, errSlots := CatchSlots(sctx, CatchError, result, ""); len(errSlots) > 0 {
		keep.AddMany(errSlots)
	}
	return keep
}

func allSlots(n uint32) []uint32 {
	slots := make([]uint32, n)
	for k := range slots {
		slots[k] = uint32(k)
	}
	return slots
}

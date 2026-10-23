package op

import (
	"github.com/superdb/super/vector"
)

// withSideErrors returns the values of vec at the slots in keep merged, in
// slot order, with the side errors errs, which belong at the slots in
// errSlots.  keep and errSlots must be sorted and disjoint.
func withSideErrors(vec vector.Any, keep []uint32, errs vector.Any, errSlots []uint32) vector.Any {
	if len(errSlots) == 0 {
		if len(keep) == int(vec.Len()) {
			return vec
		}
		return vector.Pick(vec, keep)
	}
	if len(keep) == 0 {
		return errs
	}
	tags := make([]uint32, 0, len(keep)+len(errSlots))
	i, j := 0, 0
	for i < len(keep) && j < len(errSlots) {
		if keep[i] < errSlots[j] {
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
	for ; j < len(errSlots); j++ {
		tags = append(tags, 1)
	}
	return vector.NewDynamic(tags, []vector.Any{vector.Pick(vec, keep), errs})
}

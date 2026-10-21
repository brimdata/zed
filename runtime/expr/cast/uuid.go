package cast

import (
	"uuid"

	"github.com/superdb/super/vector"
)

// castToUUID casts strings in the standard 8-4-4-4-12 hex form and 16-byte
// bytes values to uuid.
func castToUUID(vec vector.Any, index []uint32) (vector.Any, []uint32, string, bool) {
	var parse func(uint32) (uuid.UUID, bool)
	switch vec := vec.(type) {
	case *vector.UUID:
		if index == nil {
			return vec, nil, "", true
		}
		return vector.Pick(vec, index), nil, "", true
	case *vector.String:
		parse = func(idx uint32) (uuid.UUID, bool) {
			u, err := uuid.Parse(vec.Value(idx))
			return u, err == nil
		}
	case *vector.Bytes:
		parse = func(idx uint32) (uuid.UUID, bool) {
			b := vec.Value(idx)
			if len(b) != 16 {
				return uuid.UUID{}, false
			}
			return uuid.UUID(b), true
		}
	default:
		return nil, nil, "", false
	}
	n := lengthOf(vec, index)
	var vals []uuid.UUID
	var errs []uint32
	for i := range n {
		idx := i
		if index != nil {
			idx = index[i]
		}
		u, ok := parse(idx)
		if !ok {
			errs = append(errs, i)
			continue
		}
		vals = append(vals, u)
	}
	return vector.NewUUID(vals), errs, "", true
}

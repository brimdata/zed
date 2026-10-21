package cast

import (
	"github.com/superdb/super/vector"
)

func castToBytes(vec vector.Any, index []uint32) (vector.Any, []uint32, string, bool) {
	var out vector.Any
	switch vec := vec.(type) {
	case *vector.Bytes:
		out = vec
	case *vector.String:
		out = vector.NewBytes(vec.Table())
	case *vector.UUID:
		n := lengthOf(vec, index)
		bytes := make([]byte, 0, 16*n)
		offs := make([]uint32, 0, n+1)
		offs = append(offs, 0)
		for i := range n {
			idx := i
			if index != nil {
				idx = index[i]
			}
			bytes = append(bytes, vec.Values[idx][:]...)
			offs = append(offs, uint32(len(bytes)))
		}
		return vector.NewBytes(vector.NewBytesTable(offs, bytes)), nil, "", true
	default:
		return nil, nil, "", false
	}
	if index != nil {
		out = vector.Pick(out, index)
	}
	return out, nil, "", true
}

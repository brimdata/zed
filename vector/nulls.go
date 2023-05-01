package vector

type Nullmask []byte //XXX change to uint64

func NewNullmask(slots []uint32, nvals int) Nullmask {
	var nulls Nullmask
	if len(slots) > 0 {
		nulls = make([]byte, (nvals+7)/8)
		for _, slot := range slots {
			nulls[slot>>3] |= 1 << (slot & 7)
		}
	}
	return nulls
}

func (n Nullmask) Has(slot uint32) bool {
	off := slot / 8
	if off >= uint32(len(n)) {
		return false
	}
	pos := slot & 7
	return (n[off] & (1 << pos)) != 0
}

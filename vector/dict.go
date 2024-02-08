package vector

type Dict struct {
	Any
	Index  []byte
	Counts []uint32
}

var _ Any = (*Dict)(nil)

func NewDict(vals Any, index []byte, counts []uint32) *Dict {
	return &Dict{vals, index, counts}
}

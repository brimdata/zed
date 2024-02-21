package vector

import (
	"github.com/brimdata/zed/zcode"
)

type Dict struct {
	Any
	Index  []byte
	Counts []uint32
	Nulls  *Bool
}

var _ Any = (*Dict)(nil)

func NewDict(vals Any, index []byte, counts []uint32, nulls *Bool) *Dict {
	return &Dict{vals, index, counts, nulls}
}

func (d *Dict) Len() uint32 {
	return uint32(len(d.Index))
}

func (d *Dict) Serialize(builder *zcode.Builder, slot uint32) {
	if d.Nulls != nil && d.Nulls.Value(slot) {
		builder.Append(nil)
	} else {
		d.Any.Serialize(builder, uint32(d.Index[slot]))
	}
}

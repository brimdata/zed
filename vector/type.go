package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type TypeValue struct {
	Offsets []uint32
	Bytes   []byte
	Nulls   *Bool
}

var _ Any = (*TypeValue)(nil)

func NewTypeValue(offs []uint32, bytes []byte, nulls *Bool) *TypeValue {
	return &TypeValue{Offsets: offs, Bytes: bytes, Nulls: nulls}
}

func (t *TypeValue) Type() zed.Type {
	return zed.TypeType
}

func (t *TypeValue) Len() uint32 {
	return uint32(len(t.Offsets) - 1)
}

func (t *TypeValue) Value(slot uint32) []byte {
	return t.Bytes[t.Offsets[slot]:t.Offsets[slot+1]]
}

func (t *TypeValue) Serialize(b *zcode.Builder, slot uint32) {
	if t.Nulls != nil && t.Nulls.Value(slot) {
		b.Append(nil)
	} else {
		b.Append(t.Value(slot))
	}
}

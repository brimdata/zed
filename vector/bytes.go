package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Bytes struct {
	Offs  []uint32
	Bytes []byte
	Nulls *Bool
}

var _ Any = (*Bytes)(nil)

func NewBytes(offs []uint32, bytes []byte, nulls *Bool) *Bytes {
	return &Bytes{Offs: offs, Bytes: bytes, Nulls: nulls}
}

func (b *Bytes) Type() zed.Type {
	return zed.TypeBytes
}

func (b *Bytes) Len() uint32 {
	return uint32(len(b.Offs) - 1)
}

func (b *Bytes) Serialize(builder *zcode.Builder, slot uint32) {
	builder.Append(b.Value(slot))
}

func (b *Bytes) Value(slot uint32) []byte {
	if b.Nulls != nil && b.Nulls.Value(slot) {
		return nil
	}
	return b.Bytes[b.Offs[slot]:b.Offs[slot+1]]
}

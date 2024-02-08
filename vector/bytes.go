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

func (b *Bytes) Value(slot uint32) []byte {
	return b.Bytes[b.Offs[slot]:b.Offs[slot+1]]
}

func (b *Bytes) Serialize(builder *zcode.Builder, slot uint32) {
	if b.Nulls != nil && b.Nulls.Value(slot) {
		builder.Append(nil)
	} else {
		bytes := b.Bytes[b.Offs[slot]:b.Offs[slot+1]]
		builder.Append(zed.EncodeBytes(bytes))
	}
}

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
	if b.Nulls != nil && b.Nulls.Value(slot) {
		builder.Append(nil)
	} else {
		bytes := b.Bytes[b.Offs[slot]:b.Offs[slot+1]]
		builder.Append(zed.EncodeBytes(bytes))
	}
}

type DictBytes struct {
	Tags   []byte
	Offs   []uint32
	Bytes  []byte
	Counts []uint32
	Nulls  *Bool
}

var _ Any = (*DictBytes)(nil)

func NewDictBytes(tags []byte, offs []uint32, bytes []byte, counts []uint32, nulls *Bool) *DictBytes {
	return &DictBytes{Tags: tags, Offs: offs, Bytes: bytes, Counts: counts, Nulls: nulls}
}

func (d *DictBytes) Type() zed.Type {
	return zed.TypeBytes
}

func (d *DictBytes) Len() uint32 {
	return uint32(len(d.Tags))
}

func (d *DictBytes) Value(slot uint32) []byte {
	tag := d.Tags[slot]
	return d.Bytes[d.Offs[tag]:d.Offs[tag+1]]
}

func (d *DictBytes) Serialize(b *zcode.Builder, slot uint32) {
	if d.Nulls != nil && d.Nulls.Value(slot) {
		b.Append(nil)
	} else {
		b.Append(zed.EncodeBytes(d.Value(slot)))
	}
}

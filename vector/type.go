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

type DictTypeValue struct {
	Tags   []byte
	Offs   []uint32
	Bytes  []byte
	Counts []uint32
	Nulls  *Bool
}

var _ Any = (*DictTypeValue)(nil)

func NewDictTypeValue(tags []byte, offs []uint32, bytes []byte, counts []uint32, nulls *Bool) *DictTypeValue {
	return &DictTypeValue{Tags: tags, Offs: offs, Bytes: bytes, Counts: counts, Nulls: nulls}
}

func (d *DictTypeValue) Type() zed.Type {
	return zed.TypeType
}

func (d *DictTypeValue) Len() uint32 {
	return uint32(len(d.Tags))
}

func (d *DictTypeValue) Value(slot uint32) []byte {
	tag := d.Tags[slot]
	return d.Bytes[d.Offs[tag]:d.Offs[tag+1]]
}

func (d *DictTypeValue) Serialize(b *zcode.Builder, slot uint32) {
	if d.Nulls != nil && d.Nulls.Value(slot) {
		b.Append(nil)
	} else {
		b.Append(d.Value(slot))
	}
}

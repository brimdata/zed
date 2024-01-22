package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Int struct {
	Typ    zed.Type
	Values []int64
	Nulls  *Bool
}

var _ Any = (*Int)(nil)

func NewInt(typ zed.Type, values []int64, nulls *Bool) *Int {
	return &Int{Typ: typ, Values: values, Nulls: nulls}
}

func (i *Int) Type() zed.Type {
	return i.Typ
}

func (i *Int) Len() uint32 {
	return uint32(len(i.Values))
}

func (i *Int) Serialize(b *zcode.Builder, slot uint32) {
	if i.Nulls != nil && i.Nulls.Value(slot) {
		b.Append(nil)
	} else {
		b.Append(zed.EncodeInt(i.Values[slot]))
	}
}

type DictInt struct {
	Typ    zed.Type
	Tags   []byte
	Values []int64
	Counts []uint32
	Nulls  *Bool
}

var _ Any = (*DictInt)(nil)

func NewDictInt(typ zed.Type, tags []byte, values []int64, counts []uint32, nulls *Bool) *DictInt {
	return &DictInt{Typ: typ, Tags: tags, Values: values, Counts: counts, Nulls: nulls}
}

func (d *DictInt) Type() zed.Type {
	return d.Typ
}

func (d *DictInt) Len() uint32 {
	return uint32(len(d.Tags))
}

func (d *DictInt) Value(slot uint32) int64 {
	return d.Values[d.Tags[slot]]
}

func (d *DictInt) Serialize(b *zcode.Builder, slot uint32) {
	if d.Nulls != nil && d.Nulls.Value(slot) {
		b.Append(nil)
	} else {
		b.Append(zed.EncodeInt(d.Value(slot)))
	}
}

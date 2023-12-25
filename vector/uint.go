package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Uint struct {
	Typ    zed.Type
	Values []uint64
	Nulls  *Bool
}

var _ Any = (*Uint)(nil)

func NewUint(typ zed.Type, values []uint64, nulls *Bool) *Uint {
	return &Uint{Typ: typ, Values: values, Nulls: nulls}
}

func (u *Uint) Type() zed.Type {
	return u.Typ
}

func (u *Uint) Len() uint32 {
	return uint32(len(u.Values))
}

func (u *Uint) Serialize(b *zcode.Builder, slot uint32) {
	if u.Nulls != nil && u.Nulls.Value(slot) {
		b.Append(nil)
	} else {
		b.Append(zed.EncodeUint(u.Values[slot]))
	}
}

type DictUint struct {
	Typ    zed.Type
	Tags   []byte
	Values []uint64
	Counts []uint32
	Nulls  *Bool
}

var _ Any = (*DictUint)(nil)

func NewDictUint(typ zed.Type, tags []byte, values []uint64, counts []uint32, nulls *Bool) *DictUint {
	return &DictUint{Typ: typ, Tags: tags, Values: values, Counts: counts, Nulls: nulls}
}

func (d *DictUint) Type() zed.Type {
	return d.Typ
}

func (d *DictUint) Len() uint32 {
	return uint32(len(d.Tags))
}

func (d *DictUint) Value(slot uint32) uint64 {
	return d.Values[d.Tags[slot]]
}

func (d *DictUint) Serialize(b *zcode.Builder, slot uint32) {
	if d.Nulls != nil && d.Nulls.Value(slot) {
		b.Append(nil)
	} else {
		b.Append(zed.EncodeUint(d.Value(slot)))
	}
}

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
var _ Promotable = (*Uint)(nil)

func NewUint(typ zed.Type, values []uint64, nulls *Bool) *Uint {
	return &Uint{Typ: typ, Values: values, Nulls: nulls}
}

func NewUintEmpty(typ zed.Type, length uint32, nulls *Bool) *Uint {
	return NewUint(typ, make([]uint64, 0, length), nulls)
}

func (u *Uint) Append(v uint64) {
	u.Values = append(u.Values, v)
}

func (u *Uint) Type() zed.Type {
	return u.Typ
}

func (u *Uint) Len() uint32 {
	return uint32(len(u.Values))
}

func (u *Uint) Value(slot uint32) uint64 {
	return u.Values[slot]
}

func (u *Uint) Serialize(b *zcode.Builder, slot uint32) {
	if u.Nulls.Value(slot) {
		b.Append(nil)
	} else {
		b.Append(zed.EncodeUint(u.Values[slot]))
	}
}

func (u *Uint) Promote(typ zed.Type) Promotable {
	return &Uint{typ, u.Values, u.Nulls}
}

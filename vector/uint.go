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
var _ Promotable = (*Int)(nil)

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

func (u *Uint) Promote(typ zed.Type) Promotable {
	return &Uint{typ, u.Values, u.Nulls}
}

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
var _ Promotable = (*Int)(nil)

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

func (i *Int) Promote(typ zed.Type) Promotable {
	return &Int{typ, i.Values, i.Nulls}
}

package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

// XXX this should be called TypeValue no?
type Type struct {
	mem
	Typ    zed.Type
	Values []zed.Type
}

var _ Any = (*Type)(nil)

func NewType(typ zed.Type, vals []zed.Type) *Type {
	return &Type{Typ: typ, Values: vals}
}

func (t *Type) Type() zed.Type {
	return t.Typ
}

func (t *Type) NewBuilder() Builder {
	var off int
	return func(b *zcode.Builder) bool {
		if off >= len(t.Values) {
			return false
		}
		b.Append(zed.EncodeTypeValue(t.Values[off]))
		off++
		return true
	}
}

func (t *Type) Key(b []byte, slot int) []byte {
	return zed.AppendTypeValue(b, t.Values[slot])
}

func (t *Type) Length() int {
	return len(t.Values)
}

func (t *Type) Serialize(slot int) *zed.Value {
	return zed.NewValue(t.Typ, zed.EncodeTypeValue(t.Values[slot]))
}

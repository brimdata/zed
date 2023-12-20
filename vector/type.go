package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

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

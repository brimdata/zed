package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type String struct {
	mem
	Typ    zed.Type
	Values []string
}

var _ Any = (*String)(nil)

func NewString(typ zed.Type, vals []string) *String {
	return &String{Typ: typ, Values: vals}
}

func (s *String) Type() zed.Type {
	return s.Typ
}

func (s *String) NewBuilder() Builder {
	vals := s.Values
	var voff int
	return func(b *zcode.Builder) bool {
		if voff < len(vals) {
			b.Append(zed.EncodeString(vals[voff]))
			voff++
			return true

		}
		return false
	}
}

func (s *String) Key(b []byte, slot int) []byte {
	return append(b, []byte(s.Values[slot])...)
}

func (s *String) Length() int {
	return len(s.Values)
}

func (s *String) Serialize(slot int) *zed.Value {
	return zed.NewValue(s.Typ, zed.EncodeString(s.Values[slot]))
}

package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type String struct {
	mem
	Typ    zed.Type
	Values []string
	Nulls  Nullmask
}

var _ Any = (*String)(nil)

func NewString(typ zed.Type, vals []string, nulls Nullmask) *String {
	return &String{Typ: typ, Values: vals, Nulls: nulls}
}

func (s *String) Type() zed.Type {
	return s.Typ
}

func (s *String) NewBuilder() Builder {
	vals := s.Values
	nulls := s.Nulls
	var voff int
	return func(b *zcode.Builder) bool {
		if voff < len(vals) {
			if !nulls.Has(uint32(voff)) {
				b.Append(zed.EncodeString(vals[voff]))
			} else {
				b.Append(nil)
			}
			voff++
			return true

		}
		return false
	}
}

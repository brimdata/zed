package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Int struct {
	mem
	Typ    zed.Type
	Values []int64
}

var _ Any = (*Int)(nil)

func NewInt(typ zed.Type, values []int64) *Int {
	return &Int{Typ: typ, Values: values}
}

func (i *Int) Type() zed.Type {
	return i.Typ
}

func (i *Int) NewBuilder() Builder {
	vals := i.Values
	var voff int
	return func(b *zcode.Builder) bool {
		if voff < len(vals) {
			b.Append(zed.EncodeInt(vals[voff]))
			voff++
			return true
		}
		return false
	}
}

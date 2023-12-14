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

func (i *Int) Key(b []byte, slot int) []byte {
	val := i.Values[slot]
	b = append(b, byte(val>>(8*7)))
	b = append(b, byte(val>>(8*6)))
	b = append(b, byte(val>>(8*5)))
	b = append(b, byte(val>>(8*4)))
	b = append(b, byte(val>>(8*3)))
	b = append(b, byte(val>>(8*2)))
	b = append(b, byte(val>>(8*1)))
	return append(b, byte(val>>(8*0)))
}

func (i *Int) Length() int {
	return len(i.Values)
}

func (i *Int) Serialize(slot int) *zed.Value {
	return zed.NewValue(i.Typ, zed.EncodeInt(i.Values[slot]))
}

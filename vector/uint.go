package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Uint struct {
	mem
	Typ    zed.Type
	Values []uint64
}

var _ Any = (*Uint)(nil)

func NewUint(typ zed.Type, values []uint64) *Uint {
	return &Uint{Typ: typ, Values: values}
}

func (u *Uint) Type() zed.Type {
	return u.Typ
}

func (u *Uint) NewBuilder() Builder {
	vals := u.Values
	var voff int
	return func(b *zcode.Builder) bool {
		if voff < len(vals) {
			b.Append(zed.EncodeUint(vals[voff]))
			voff++
			return true
		}
		return false
	}
}

func (u *Uint) Key(b []byte, slot int) []byte {
	val := u.Values[slot]
	b = append(b, byte(val>>(8*7)))
	b = append(b, byte(val>>(8*6)))
	b = append(b, byte(val>>(8*5)))
	b = append(b, byte(val>>(8*4)))
	b = append(b, byte(val>>(8*3)))
	b = append(b, byte(val>>(8*2)))
	b = append(b, byte(val>>(8*1)))
	return append(b, byte(val>>(8*0)))
}

func (u *Uint) Length() int {
	return len(u.Values)
}

func (u *Uint) Serialize(slot int) *zed.Value {
	return zed.NewValue(u.Typ, zed.EncodeUint(u.Values[slot]))
}

package vector

import (
	"math"
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Float struct {
	mem
	Typ    zed.Type
	Values []float64
}

var _ Any = (*Float)(nil)

func NewFloat(typ zed.Type, values []float64) *Float {
	return &Float{Typ: typ, Values: values}
}

func (f *Float) Type() zed.Type {
	return f.Typ
}

func (f *Float) NewBuilder() Builder {
	typeID := f.Typ.ID()
	var off int
	return func(b *zcode.Builder) bool {
		if off >= len(f.Values) {
			return false
		}
		switch typeID {
		case zed.IDFloat16:
			b.Append(zed.EncodeFloat16(float32(f.Values[off])))
		case zed.IDFloat32:
			b.Append(zed.EncodeFloat32(float32(f.Values[off])))
		case zed.IDFloat64:
			b.Append(zed.EncodeFloat64(f.Values[off]))
		default:
			panic(f.Typ)
		}
		off++
		return true
	}
}

func (f *Float) Key(b []byte, slot int) []byte {
	val := math.Float64bits(f.Values[slot])
	b = append(b, byte(val>>(8*7)))
	b = append(b, byte(val>>(8*6)))
	b = append(b, byte(val>>(8*5)))
	b = append(b, byte(val>>(8*4)))
	b = append(b, byte(val>>(8*3)))
	b = append(b, byte(val>>(8*2)))
	b = append(b, byte(val>>(8*1)))
	return append(b, byte(val>>(8*0)))
}

func (f *Float) Length() int {
	return len(f.Values)
}

func (f *Float) Serialize(slot int) *zed.Value {
	return zed.NewFloat64(f.Values[slot])
}

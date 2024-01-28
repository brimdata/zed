package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Float struct {
	Typ    zed.Type
	Values []float64
	Nulls  *Bool
}

var _ Any = (*Float)(nil)

func NewFloat(typ zed.Type, values []float64, nulls *Bool) *Float {
	return &Float{Typ: typ, Values: values, Nulls: nulls}
}

func (f *Float) Type() zed.Type {
	return f.Typ
}

func (f *Float) Len() uint32 {
	return uint32(len(f.Values))
}

func (f *Float) Serialize(b *zcode.Builder, slot uint32) {
	if f.Nulls != nil && f.Nulls.Value(slot) {
		b.Append(nil)
		return
	}
	switch f.Typ.ID() {
	case zed.IDFloat16:
		b.Append(zed.EncodeFloat16(float32(f.Values[slot])))
	case zed.IDFloat32:
		b.Append(zed.EncodeFloat32(float32(f.Values[slot])))
	case zed.IDFloat64:
		b.Append(zed.EncodeFloat64(f.Values[slot]))
	default:
		panic(f.Typ)
	}
}

type DictFloat struct {
	Typ    zed.Type
	Tags   []byte
	Values []float64
	Counts []uint32
	Nulls  *Bool
}

var _ Any = (*DictFloat)(nil)

func NewDictFloat(typ zed.Type, tags []byte, values []float64, counts []uint32, nulls *Bool) *DictFloat {
	return &DictFloat{Typ: typ, Tags: tags, Values: values, Counts: counts, Nulls: nulls}
}

func (d *DictFloat) Type() zed.Type {
	return d.Typ
}

func (d *DictFloat) Len() uint32 {
	return uint32(len(d.Tags))
}

func (d *DictFloat) Value(slot uint32) float64 {
	return d.Values[d.Tags[slot]]
}

func (d *DictFloat) Serialize(b *zcode.Builder, slot uint32) {
	if d.Nulls != nil && d.Nulls.Value(slot) {
		b.Append(nil)
		return
	}
	switch d.Typ.ID() {
	case zed.IDFloat16:
		b.Append(zed.EncodeFloat16(float32(d.Value(slot))))
	case zed.IDFloat32:
		b.Append(zed.EncodeFloat32(float32(d.Value(slot))))
	case zed.IDFloat64:
		b.Append(zed.EncodeFloat64(d.Value(slot)))
	default:
		panic(d.Typ)
	}
}

func (d *DictFloat) Unravel() *Float {
	n := len(d.Tags)
	out := make([]float64, n)
	for k := 0; k < n; k++ {
		out[k] = d.Values[d.Tags[k]]
	}
	return NewFloat(d.Typ, out, d.Nulls)
}

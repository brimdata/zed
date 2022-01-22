package zed

import (
	"encoding/binary"
	"math"

	"github.com/brimdata/zed/zcode"
)

func DecodeFloat(zb zcode.Bytes) float64 {
	if zb == nil {
		return 0
	}
	switch len(zb) {
	case 4:
		bits := binary.LittleEndian.Uint32(zb)
		return float64(math.Float32frombits(bits))
	case 8:
		bits := binary.LittleEndian.Uint64(zb)
		return math.Float64frombits(bits)
	}
	panic("float encoding is neither 4 nor 8 bytes")
}

type TypeOfFloat32 struct{}

func NewFloat32(f float32) *Value {
	return &Value{TypeFloat32, EncodeFloat32(f)}
}

func AppendFloat32(zb zcode.Bytes, f float32) zcode.Bytes {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, math.Float32bits(f))
	return append(zb, buf...)
}

func EncodeFloat32(d float32) zcode.Bytes {
	return AppendFloat32(nil, d)
}

func DecodeFloat32(zb zcode.Bytes) float32 {
	if zb == nil {
		return 0
	}
	return math.Float32frombits(binary.LittleEndian.Uint32(zb))
}

func (t *TypeOfFloat32) ID() int {
	return IDFloat32
}

func (t *TypeOfFloat32) Kind() string {
	return "primitive"
}

func (t *TypeOfFloat32) Marshal(zb zcode.Bytes) interface{} {
	return DecodeFloat32(zb)
}

type TypeOfFloat64 struct{}

func NewFloat64(f float64) *Value {
	return &Value{TypeFloat64, EncodeFloat64(f)}
}

func AppendFloat64(zb zcode.Bytes, d float64) zcode.Bytes {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, math.Float64bits(d))
	return append(zb, buf...)
}

func EncodeFloat64(d float64) zcode.Bytes {
	return AppendFloat64(nil, d)
}

func DecodeFloat64(zv zcode.Bytes) float64 {
	if zv == nil {
		return 0
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(zv))
}

func (t *TypeOfFloat64) ID() int {
	return IDFloat64
}

func (t *TypeOfFloat64) Kind() string {
	return "primitive"
}

func (t *TypeOfFloat64) Marshal(zv zcode.Bytes) interface{} {
	return DecodeFloat64(zv)
}

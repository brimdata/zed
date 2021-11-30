package zed

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strconv"

	"github.com/brimdata/zed/zcode"
)

func DecodeFloat(zb zcode.Bytes) (float64, error) {
	switch len(zb) {
	case 4:
		bits := binary.LittleEndian.Uint32(zb)
		return float64(math.Float32frombits(bits)), nil
	case 8:
		bits := binary.LittleEndian.Uint64(zb)
		return math.Float64frombits(bits), nil
	}
	return 0, errors.New("float encoding is neither 4 nor 8 bytes")
}

type TypeOfFloat32 struct{}

func NewFloat32(f float32) Value {
	return Value{TypeFloat32, EncodeFloat32(f)}
}

func AppendFloat32(zb zcode.Bytes, f float32) zcode.Bytes {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, math.Float32bits(f))
	return append(zb, buf...)
}

func EncodeFloat32(d float32) zcode.Bytes {
	return AppendFloat32(nil, d)
}

func DecodeFloat32(zb zcode.Bytes) (float32, error) {
	if len(zb) != 4 {
		return 0, errors.New("byte encoding of float32 not 4 bytes")
	}
	bits := binary.LittleEndian.Uint32(zb)
	return math.Float32frombits(bits), nil
}

func (t *TypeOfFloat32) ID() int {
	return IDFloat32
}

func (t *TypeOfFloat32) String() string {
	return "float32"
}

func (t *TypeOfFloat32) Marshal(zb zcode.Bytes) (interface{}, error) {
	return DecodeFloat32(zb)
}

func (t *TypeOfFloat32) Format(zb zcode.Bytes) string {
	f, err := DecodeFloat32(zb)
	if err != nil {
		return badZNG(err, t, zb)
	}
	if f == float32(int64(f)) {
		return fmt.Sprintf("%d.", int(f))
	}
	return strconv.FormatFloat(float64(f), 'g', -1, 32)
}

type TypeOfFloat64 struct{}

func NewFloat64(f float64) Value {
	return Value{TypeFloat64, EncodeFloat64(f)}
}

func AppendFloat64(zb zcode.Bytes, d float64) zcode.Bytes {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, math.Float64bits(d))
	return append(zb, buf...)
}

func EncodeFloat64(d float64) zcode.Bytes {
	return AppendFloat64(nil, d)
}

func DecodeFloat64(zv zcode.Bytes) (float64, error) {
	if len(zv) != 8 {
		return 0, errors.New("byte encoding of double not 8 bytes")
	}
	bits := binary.LittleEndian.Uint64(zv)
	return math.Float64frombits(bits), nil
}

func (t *TypeOfFloat64) ID() int {
	return IDFloat64
}

func (t *TypeOfFloat64) String() string {
	return "float64"
}

func (t *TypeOfFloat64) Marshal(zv zcode.Bytes) (interface{}, error) {
	return DecodeFloat64(zv)
}

func (t *TypeOfFloat64) Format(zv zcode.Bytes) string {
	d, err := DecodeFloat64(zv)
	if err != nil {
		return badZNG(err, t, zv)
	}
	if d == float64(int64(d)) {
		return fmt.Sprintf("%d.", int64(d))
	}
	return strconv.FormatFloat(d, 'g', -1, 64)
}

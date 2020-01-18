package zng

import (
	"encoding/binary"
	"errors"
	"math"
	"strconv"

	"github.com/mccanne/zq/zcode"
)

type TypeOfDouble struct{}

var compareFloat = map[string]func(float64, float64) bool{
	"eql":  func(a, b float64) bool { return a == b },
	"neql": func(a, b float64) bool { return a != b },
	"gt":   func(a, b float64) bool { return a > b },
	"gte":  func(a, b float64) bool { return a >= b },
	"lt":   func(a, b float64) bool { return a < b },
	"lte":  func(a, b float64) bool { return a <= b }}

func NewDouble(f float64) Value {
	return Value{TypeDouble, EncodeDouble(f)}
}

func EncodeDouble(d float64) zcode.Bytes {
	bits := math.Float64bits(d)
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], bits)
	return b[:]
}

func DecodeDouble(zv zcode.Bytes) (float64, error) {
	if len(zv) != 8 {
		return 0, errors.New("byte encoding of double not 8 bytes")
	}
	bits := binary.LittleEndian.Uint64(zv)
	return math.Float64frombits(bits), nil
}

func (t *TypeOfDouble) Parse(in []byte) (zcode.Bytes, error) {
	d, err := UnsafeParseFloat64(in)
	if err != nil {
		return nil, err
	}
	return EncodeDouble(d), nil
}

func (t *TypeOfDouble) Id() int {
	return IdFloat64
}

func (t *TypeOfDouble) String() string {
	return "double"
}

func (t *TypeOfDouble) StringOf(zv zcode.Bytes) string {
	d, err := DecodeDouble(zv)
	if err != nil {
		return badZng(err, t, zv)
	}
	return strconv.FormatFloat(d, 'f', -1, 64)
}

func (t *TypeOfDouble) Marshal(zv zcode.Bytes) (interface{}, error) {
	return DecodeDouble(zv)
}

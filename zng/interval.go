package zng

import (
	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/zcode"
)

type TypeOfInterval struct{}

func NewInterval(i int64) Value {
	return Value{TypeInterval, EncodeInterval(i)}
}

func EncodeInterval(i int64) zcode.Bytes {
	return EncodeInt(i)
}

func DecodeInterval(zv zcode.Bytes) (int64, error) {
	return DecodeInt(zv)
}

func (t *TypeOfInterval) Parse(in []byte) (zcode.Bytes, error) {
	dur, err := nano.ParseDuration(in)
	if err != nil {
		return nil, err
	}
	return EncodeInterval(int64(dur)), nil
}

func (t *TypeOfInterval) ID() int {
	return IdDuration
}

func (t *TypeOfInterval) String() string {
	return "interval"
}

func (t *TypeOfInterval) StringOf(zv zcode.Bytes) string {
	i, err := DecodeInterval(zv)
	if err != nil {
		return badZng(err, t, zv)
	}
	// This format of a fractional second is used by zeek in logs.
	// It uses enough precision to fully represent the 64-bit ns
	// accuracy of a nano Duration. Such values cannot be represented by
	// float64's without loss of the least significant digits of ns,
	return nano.DurationString(i)
}

func (t *TypeOfInterval) Marshal(zv zcode.Bytes) (interface{}, error) {
	return DecodeInterval(zv)
}

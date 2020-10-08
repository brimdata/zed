package zng

import (
	"github.com/brimsec/zq/alpha/zcode"
	"github.com/brimsec/zq/pkg/nano"
)

type TypeOfTime struct{}

func NewTime(ts nano.Ts) Value {
	return Value{TypeTime, EncodeTime(ts)}
}

func EncodeTime(t nano.Ts) zcode.Bytes {
	var b [8]byte
	n := zcode.EncodeCountedVarint(b[:], int64(t))
	return b[:n]
}

func DecodeTime(zv zcode.Bytes) (nano.Ts, error) {
	if zv == nil {
		return 0, ErrUnset
	}
	return nano.Ts(zcode.DecodeCountedVarint(zv)), nil
}

func (t *TypeOfTime) Parse(in []byte) (zcode.Bytes, error) {
	ts, err := nano.Parse(in)
	if err != nil {
		return nil, err
	}
	return EncodeTime(ts), nil
}

func (t *TypeOfTime) ID() int {
	return IdTime
}

func (t *TypeOfTime) String() string {
	return "time"
}

func (t *TypeOfTime) StringOf(zv zcode.Bytes, _ OutFmt, _ bool) string {
	ts, err := DecodeTime(zv)
	if err != nil {
		return badZng(err, t, zv)
	}
	// This format of a fractional second is used by zeek in logs.
	// It uses enough precision to fully represent the 64-bit ns
	// accuracy of a nano.Ts.  Such values cannot be representd by
	// float64's without loss of the least significant digits of ns,
	return ts.StringFloat()
}

func (t *TypeOfTime) Marshal(zv zcode.Bytes) (interface{}, error) {
	ts, err := DecodeTime(zv)
	if err != nil {
		return nil, err
	}
	return ts.StringFloat(), nil
}

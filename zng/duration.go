package zng

import (
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zcode"
)

type TypeOfDuration struct{}

func NewDuration(d nano.Duration) Value {
	return Value{TypeDuration, EncodeDuration(d)}
}

func EncodeDuration(d nano.Duration) zcode.Bytes {
	return EncodeInt(int64(d))
}

func AppendDuration(bytes zcode.Bytes, d nano.Duration) zcode.Bytes {
	return AppendInt(bytes, int64(d))
}

func DecodeDuration(zv zcode.Bytes) (nano.Duration, error) {
	i, err := DecodeInt(zv)
	if err != nil {
		return 0, err
	}
	return nano.Duration(i), nil
}

func (t *TypeOfDuration) ID() int {
	return IdDuration
}

func (t *TypeOfDuration) String() string {
	return "duration"
}

func (t *TypeOfDuration) Marshal(zv zcode.Bytes) (interface{}, error) {
	return t.ZSONOf(zv), nil
}

func (t *TypeOfDuration) ZSON() string {
	return "duration"
}

func (t *TypeOfDuration) ZSONOf(zv zcode.Bytes) string {
	d, err := DecodeDuration(zv)
	if err != nil {
		return badZng(err, t, zv)
	}
	return d.String()
}

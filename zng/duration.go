package zng

import (
	"time"

	"github.com/brimsec/zq/zcode"
)

type TypeOfDuration struct{}

func NewDuration(i int64) Value {
	return Value{TypeDuration, EncodeDuration(i)}
}

func EncodeDuration(i int64) zcode.Bytes {
	return EncodeInt(i)
}

func AppendDuration(bytes zcode.Bytes, d int64) zcode.Bytes {
	return AppendInt(bytes, d)
}

func DecodeDuration(zv zcode.Bytes) (int64, error) {
	return DecodeInt(zv)
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
	ns, err := DecodeDuration(zv)
	if err != nil {
		return badZng(err, t, zv)
	}
	return time.Duration(ns).String()
}

package zed

import (
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zcode"
)

type TypeOfDuration struct{}

func NewDuration(d nano.Duration) *Value {
	return &Value{TypeDuration, EncodeDuration(d)}
}

func EncodeDuration(d nano.Duration) zcode.Bytes {
	return EncodeInt(int64(d))
}

func AppendDuration(bytes zcode.Bytes, d nano.Duration) zcode.Bytes {
	return AppendInt(bytes, int64(d))
}

func DecodeDuration(zv zcode.Bytes) nano.Duration {
	return nano.Duration(DecodeInt(zv))
}

func (t *TypeOfDuration) ID() int {
	return IDDuration
}

func (t *TypeOfDuration) Kind() Kind {
	return PrimitiveKind
}

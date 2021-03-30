package result

import (
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zng"
)

type Buffer zcode.Bytes

func (b *Buffer) Int(v int64) zcode.Bytes {
	*b = Buffer(zng.AppendInt(zcode.Bytes((*b)[:0]), v))
	return zcode.Bytes(*b)
}

func (b *Buffer) Uint(v uint64) zcode.Bytes {
	*b = Buffer(zng.AppendUint(zcode.Bytes((*b)[:0]), v))
	return zcode.Bytes(*b)
}

func (b *Buffer) Float64(v float64) zcode.Bytes {
	*b = Buffer(zng.AppendFloat64(zcode.Bytes((*b)[:0]), v))
	return zcode.Bytes(*b)
}

func (b *Buffer) Time(v nano.Ts) zcode.Bytes {
	*b = Buffer(zng.AppendTime(zcode.Bytes((*b)[:0]), v))
	return zcode.Bytes(*b)
}

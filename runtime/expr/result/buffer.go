package result

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zcode"
)

type Buffer zcode.Bytes

func (b *Buffer) Int(v int64) zcode.Bytes {
	*b = Buffer(zed.AppendInt(zcode.Bytes((*b)[:0]), v))
	return zcode.Bytes(*b)
}

func (b *Buffer) Uint(v uint64) zcode.Bytes {
	*b = Buffer(zed.AppendUint(zcode.Bytes((*b)[:0]), v))
	return zcode.Bytes(*b)
}

func (b *Buffer) Float32(v float32) zcode.Bytes {
	*b = Buffer(zed.AppendFloat32(zcode.Bytes((*b)[:0]), v))
	return zcode.Bytes(*b)
}

func (b *Buffer) Float64(v float64) zcode.Bytes {
	*b = Buffer(zed.AppendFloat64(zcode.Bytes((*b)[:0]), v))
	return zcode.Bytes(*b)
}

func (b *Buffer) Time(v nano.Ts) zcode.Bytes {
	*b = Buffer(zed.AppendTime(zcode.Bytes((*b)[:0]), v))
	return zcode.Bytes(*b)
}

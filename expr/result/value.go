package result

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/nano"
)

type Value zed.Value

func (v *Value) Int64(native int64) *zed.Value {
	v.Type = zed.TypeInt64
	v.Bytes = zed.AppendInt(v.Bytes[:0], native)
	return (*zed.Value)(v)
}

func (v *Value) Uint64(native uint64) *zed.Value {
	v.Type = zed.TypeUint64
	v.Bytes = zed.AppendUint(v.Bytes[:0], native)
	return (*zed.Value)(v)
}

func (v *Value) Float32(native float32) *zed.Value {
	v.Type = zed.TypeFloat32
	v.Bytes = zed.AppendFloat32(v.Bytes[:0], native)
	return (*zed.Value)(v)
}

func (v *Value) Float64(native float64) *zed.Value {
	v.Type = zed.TypeFloat64
	v.Bytes = zed.AppendFloat64(v.Bytes[:0], native)
	return (*zed.Value)(v)
}

func (v *Value) Duration(native nano.Duration) *zed.Value {
	v.Type = zed.TypeDuration
	v.Bytes = zed.AppendDuration(v.Bytes[:0], native)
	return (*zed.Value)(v)
}

func (v *Value) Time(native nano.Ts) *zed.Value {
	v.Type = zed.TypeTime
	v.Bytes = zed.AppendTime(v.Bytes[:0], native)
	return (*zed.Value)(v)
}

func (v *Value) Error(err error) *zed.Value {
	//XXX this clobbers the stashed byte slice
	*v = (Value)(zed.NewError(err))
	return (*zed.Value)(v)
}

func (v *Value) Copy(val *zed.Value) *zed.Value {
	v.Type = val.Type
	v.Bytes = append(v.Bytes[:0], val.Bytes...)
	return (*zed.Value)(v)
}

package result

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/nano"
)

type Value zed.Value

func (v *Value) Int64(native int64) *zed.Value {
	return v.Int(zed.TypeInt64, native)
}

func (v *Value) Int(typ zed.Type, native int64) *zed.Value {
	v.Type = typ
	v.Bytes = zed.AppendInt(v.Bytes[:0], native)
	return (*zed.Value)(v)
}

func (v *Value) Uint64(native uint64) *zed.Value {
	return v.Uint(zed.TypeUint64, native)
}

func (v *Value) Uint(typ zed.Type, native uint64) *zed.Value {
	v.Type = typ
	v.Bytes = zed.AppendUint(v.Bytes[:0], native)
	return (*zed.Value)(v)
}

func (v *Value) Float64(native float64) *zed.Value {
	return v.Float(zed.TypeFloat64, native)
}

func (v *Value) Float(typ zed.Type, native float64) *zed.Value {
	v.Type = typ
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

func (v *Value) String(native string) *zed.Value {
	if native == "" {
		return &zed.Value{zed.TypeString, []byte{}}
	}
	v.Type = zed.TypeString
	v.Bytes = append(v.Bytes[:0], []byte(native)...)
	return (*zed.Value)(v)
}

func (v *Value) Error(err error) *zed.Value {
	//XXX this clobbers the stashed byte slice
	*v = (Value)(zed.NewError(err))
	return (*zed.Value)(v)
}

func (v *Value) Errorf(format string, args ...interface{}) *zed.Value {
	*v = (Value)(zed.NewErrorf(format, args...))
	return (*zed.Value)(v)
}

func (v *Value) Copy(val *zed.Value) *zed.Value {
	if val.Bytes == nil {
		return &zed.Value{Type: val.Type}
	}
	if v.Bytes == nil {
		v.Bytes = []byte{}
	}
	v.Type = val.Type
	v.Bytes = append(v.Bytes[:0], val.Bytes...)
	return (*zed.Value)(v)
}

func (v *Value) CopyVal(val zed.Value) *zed.Value {
	if val.Bytes == nil {
		return &zed.Value{Type: val.Type}
	}
	if v.Bytes == nil {
		v.Bytes = []byte{}
	}
	v.Type = val.Type
	v.Bytes = append(v.Bytes[:0], val.Bytes...)
	return (*zed.Value)(v)
}

package function

import (
	"math"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/anymath"
	"github.com/brimdata/zed/expr/coerce"
	"github.com/brimdata/zed/zson"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#abs.md
type Abs struct{}

func (a *Abs) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	v := args[0]
	id := v.Type.ID()
	if zed.IsFloat(id) {
		f, err := zed.DecodeFloat64(v.Bytes)
		if err != nil {
			panic(err)
		}
		f = math.Abs(f)
		return newFloat64(ctx, f)
	}
	if !zed.IsInteger(id) {
		return newErrorf(ctx, "abs: not a number: %s", zson.MustFormatValue(args[0]))
	}
	if !zed.IsSigned(id) {
		return ctx.CopyValue(args[0])
	}
	x, err := zed.DecodeInt(v.Bytes)
	if err != nil {
		panic(err)
	}
	if x < 0 {
		x = -x
	}
	return newInt64(ctx, x)
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#ceil
type Ceil struct{}

func (c *Ceil) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	v := args[0]
	id := v.Type.ID()
	switch {
	case zed.IsFloat(id):
		f, err := zed.DecodeFloat64(v.Bytes)
		if err != nil {
			panic(err)
		}
		f = math.Ceil(f)
		return newFloat64(ctx, f)
	case zed.IsInteger(id):
		return ctx.CopyValue(args[0])
	default:
		return newErrorf(ctx, "ceil: not a number")
	}
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#floor
type Floor struct{}

func (f *Floor) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	v := args[0]
	id := v.Type.ID()
	switch {
	case zed.IsFloat(id):
		v, _ := zed.DecodeFloat64(v.Bytes)
		v = math.Floor(v)
		return newFloat64(ctx, v)
	case zed.IsInteger(id):
		return ctx.CopyValue(args[0])
	default:
		return newErrorf(ctx, "floor: not a number")
	}
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#log
type Log struct{}

func (l *Log) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	x, ok := coerce.ToFloat(args[0])
	if !ok {
		return newErrorf(ctx, "log: numeric argument required")
	}
	if x <= 0 {
		return newErrorf(ctx, "log: illegal argument: %s", zson.MustFormatValue(args[0]))
	}
	return newFloat64(ctx, math.Log(x))
}

type reducer struct {
	name string
	fn   *anymath.Function
}

func (r *reducer) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	zv := args[0]
	typ := zv.Type
	id := typ.ID()
	if zed.IsFloat(id) {
		//XXX this is wrong like math aggregators...
		// need to be more robust and adjust type as new types encountered
		result, _ := zed.DecodeFloat64(zv.Bytes)
		for _, val := range args[1:] {
			v, ok := coerce.ToFloat(zv)
			if !ok {
				return newErrorf(ctx, "%s: not a number: %s", r.name, zson.MustFormatValue(val))
			}
			result = r.fn.Float64(result, v)
		}
		return newFloat64(ctx, result)
	}
	if !zed.IsNumber(id) {
		return newErrorf(ctx, "%s: not a number: %s", r.name, zson.MustFormatValue(zv))
	}
	if zed.IsSigned(id) {
		result, _ := zed.DecodeInt(zv.Bytes)
		for _, val := range args[1:] {
			//XXX this is really bad because we silently coerce
			// floats to ints if we hit a float first
			v, ok := coerce.ToInt(val)
			if !ok {
				return newErrorf(ctx, "%s: not a number: %s", r.name, zson.MustFormatValue(val))
			}
			result = r.fn.Int64(result, v)
		}
		return newInt64(ctx, result)
	}
	result, _ := zed.DecodeUint(zv.Bytes)
	for _, val := range args[1:] {
		v, ok := coerce.ToUint(val)
		if !ok {
			return newErrorf(ctx, "%s: not a number: %s", r.name, zson.MustFormatValue(val))
		}
		result = r.fn.Uint64(result, v)
	}
	return newUint64(ctx, result)
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#round
type Round struct{}

func (r *Round) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	zv := args[0]
	id := zv.Type.ID()
	if zed.IsFloat(id) {
		f, err := zed.DecodeFloat64(zv.Bytes)
		if err != nil {
			panic(err)
		}
		return newFloat64(ctx, math.Round(f))
	}
	if !zed.IsNumber(id) {
		return newErrorf(ctx, "round: not a number: %s", zson.MustFormatValue(zv))
	}
	return ctx.CopyValue(args[0])
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#pow
type Pow struct{}

func (p *Pow) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	x, ok := coerce.ToFloat(args[0])
	if !ok {
		return newErrorf(ctx, "pow: not a number: %s", zson.MustFormatValue(args[0]))
	}
	y, ok := coerce.ToFloat(args[1])
	if !ok {
		return newErrorf(ctx, "pow: not a number: %s", zson.MustFormatValue(args[1]))
	}
	return newFloat64(ctx, math.Pow(x, y))
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#sqrt
type Sqrt struct{}

func (s *Sqrt) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	x, ok := coerce.ToFloat(args[0])
	if !ok {
		return newErrorf(ctx, "sqrt: not a number: %s", zson.MustFormatValue(args[0]))
	}
	return newFloat64(ctx, math.Sqrt(x))
}

package function

import (
	"math"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/anymath"
	"github.com/brimdata/zed/runtime/expr/coerce"
	"github.com/brimdata/zed/zson"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#abs.md
type Abs struct {
	zctx *zed.Context
}

func (a *Abs) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	v := args[0]
	id := v.Type.ID()
	if id == zed.IDFloat16 {
		f := math.Abs(v.Float())
		return newFloat16(ctx, float32(f))
	}
	if id == zed.IDFloat32 {
		f := math.Abs(v.Float())
		return newFloat32(ctx, float32(f))
	}
	if id == zed.IDFloat64 {
		f := math.Abs(v.Float())
		return newFloat64(ctx, f)
	}
	if !zed.IsInteger(id) {
		return newErrorf(a.zctx, ctx, "abs: not a number: %s", zson.FormatValue(&args[0]))
	}
	if !zed.IsSigned(id) {
		return ctx.CopyValue(&args[0])
	}
	x := v.Int()
	if x < 0 {
		x = -x
	}
	return ctx.CopyValue(zed.NewInt64(x))
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#ceil
type Ceil struct {
	zctx *zed.Context
}

func (c *Ceil) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	v := args[0]
	id := v.Type.ID()
	switch {
	case id == zed.IDFloat16:
		f := math.Ceil(v.Float())
		return newFloat16(ctx, float32(f))
	case id == zed.IDFloat32:
		f := math.Ceil(v.Float())
		return newFloat32(ctx, float32(f))
	case id == zed.IDFloat64:
		f := math.Ceil(v.Float())
		return newFloat64(ctx, f)
	case zed.IsInteger(id):
		return ctx.CopyValue(&args[0])
	default:
		return newErrorf(c.zctx, ctx, "ceil: not a number")
	}
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#floor
type Floor struct {
	zctx *zed.Context
}

func (f *Floor) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	v := args[0]
	id := v.Type.ID()
	switch {
	case id == zed.IDFloat16:
		v := math.Floor(v.Float())
		return newFloat16(ctx, float32(v))
	case id == zed.IDFloat32:
		v := math.Floor(v.Float())
		return newFloat32(ctx, float32(v))
	case id == zed.IDFloat64:
		v := math.Floor(v.Float())
		return newFloat64(ctx, v)
	case zed.IsInteger(id):
		return ctx.CopyValue(&args[0])
	default:
		return newErrorf(f.zctx, ctx, "floor: not a number")
	}
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#log
type Log struct {
	zctx *zed.Context
}

func (l *Log) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	x, ok := coerce.ToFloat(&args[0])
	if !ok {
		return newErrorf(l.zctx, ctx, "log: numeric argument required")
	}
	if x <= 0 {
		return newErrorf(l.zctx, ctx, "log: illegal argument: %s", zson.FormatValue(&args[0]))
	}
	return newFloat64(ctx, math.Log(x))
}

type reducer struct {
	zctx *zed.Context
	name string
	fn   *anymath.Function
}

func (r *reducer) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	val0 := &args[0]
	typ := val0.Type
	id := typ.ID()
	if zed.IsFloat(id) {
		//XXX this is wrong like math aggregators...
		// need to be more robust and adjust type as new types encountered
		result := val0.Float()
		for _, val := range args[1:] {
			v, ok := coerce.ToFloat(&val)
			if !ok {
				return newErrorf(r.zctx, ctx, "%s: not a number: %s", r.name, zson.FormatValue(&val))
			}
			result = r.fn.Float64(result, v)
		}
		return newFloat64(ctx, result)
	}
	if !zed.IsNumber(id) {
		return newErrorf(r.zctx, ctx, "%s: not a number: %s", r.name, zson.FormatValue(val0))
	}
	if zed.IsSigned(id) {
		result := val0.Int()
		for _, val := range args[1:] {
			//XXX this is really bad because we silently coerce
			// floats to ints if we hit a float first
			v, ok := coerce.ToInt(&val)
			if !ok {
				return newErrorf(r.zctx, ctx, "%s: not a number: %s", r.name, zson.FormatValue(&val))
			}
			result = r.fn.Int64(result, v)
		}
		return ctx.CopyValue(zed.NewInt64(result))
	}
	result := val0.Uint()
	for _, val := range args[1:] {
		v, ok := coerce.ToUint(&val)
		if !ok {
			return newErrorf(r.zctx, ctx, "%s: not a number: %s", r.name, zson.FormatValue(&val))
		}
		result = r.fn.Uint64(result, v)
	}
	return ctx.CopyValue(zed.NewUint64(result))
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#round
type Round struct {
	zctx *zed.Context
}

func (r *Round) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	val := &args[0]
	id := val.Type.ID()
	if id == zed.IDFloat16 {
		return newFloat16(ctx, float32(math.Round(val.Float())))
	}
	if id == zed.IDFloat32 {
		return newFloat32(ctx, float32(math.Round(val.Float())))
	}
	if id == zed.IDFloat64 {
		return newFloat64(ctx, math.Round(val.Float()))
	}
	if !zed.IsNumber(id) {
		return newErrorf(r.zctx, ctx, "round: not a number: %s", zson.FormatValue(val))
	}
	return ctx.CopyValue(&args[0])
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#pow
type Pow struct {
	zctx *zed.Context
}

func (p *Pow) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	x, ok := coerce.ToFloat(&args[0])
	if !ok {
		return newErrorf(p.zctx, ctx, "pow: not a number: %s", zson.FormatValue(&args[0]))
	}
	y, ok := coerce.ToFloat(&args[1])
	if !ok {
		return newErrorf(p.zctx, ctx, "pow: not a number: %s", zson.FormatValue(&args[1]))
	}
	return newFloat64(ctx, math.Pow(x, y))
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#sqrt
type Sqrt struct {
	zctx *zed.Context
}

func (s *Sqrt) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	x, ok := coerce.ToFloat(&args[0])
	if !ok {
		return newErrorf(s.zctx, ctx, "sqrt: not a number: %s", zson.FormatValue(&args[0]))
	}
	return newFloat64(ctx, math.Sqrt(x))
}

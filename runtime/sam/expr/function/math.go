package function

import (
	"math"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/anymath"
	"github.com/brimdata/zed/runtime/sam/expr"
	"github.com/brimdata/zed/runtime/sam/expr/coerce"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#abs.md
type Abs struct {
	zctx *zed.Context
}

func (a *Abs) Call(ectx expr.Context, args []zed.Value) zed.Value {
	val := args[0]
	switch id := val.Type().ID(); {
	case zed.IsUnsigned(id):
		return val
	case zed.IsSigned(id):
		x := val.Int()
		if x < 0 {
			x = -x
		}
		return zed.NewInt(val.Type(), x)
	case zed.IsFloat(id):
		return zed.NewFloat(val.Type(), math.Abs(val.Float()))
	}
	return a.zctx.WrapError(ectx.Arena(), "abs: not a number", val)
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#ceil
type Ceil struct {
	zctx *zed.Context
}

func (c *Ceil) Call(ectx expr.Context, args []zed.Value) zed.Value {
	val := args[0]
	switch id := val.Type().ID(); {
	case zed.IsUnsigned(id) || zed.IsSigned(id):
		return val
	case zed.IsFloat(id):
		return zed.NewFloat(val.Type(), math.Ceil(val.Float()))
	}
	return c.zctx.WrapError(ectx.Arena(), "ceil: not a number", val)
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#floor
type Floor struct {
	zctx *zed.Context
}

func (f *Floor) Call(ectx expr.Context, args []zed.Value) zed.Value {
	val := args[0]
	switch id := val.Type().ID(); {
	case zed.IsUnsigned(id) || zed.IsSigned(id):
		return val
	case zed.IsFloat(id):
		return zed.NewFloat(val.Type(), math.Floor(val.Float()))
	}
	return f.zctx.WrapError(ectx.Arena(), "floor: not a number", val)
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#log
type Log struct {
	zctx *zed.Context
}

func (l *Log) Call(ectx expr.Context, args []zed.Value) zed.Value {
	x, ok := coerce.ToFloat(args[0])
	if !ok {
		return l.zctx.WrapError(ectx.Arena(), "log: not a number", args[0])
	}
	if x <= 0 {
		return l.zctx.WrapError(ectx.Arena(), "log: illegal argument", args[0])
	}
	return zed.NewFloat64(math.Log(x))
}

type reducer struct {
	zctx *zed.Context
	name string
	fn   *anymath.Function
}

func (r *reducer) Call(ectx expr.Context, args []zed.Value) zed.Value {
	val0 := args[0]
	switch id := val0.Type().ID(); {
	case zed.IsUnsigned(id):
		result := val0.Uint()
		for _, val := range args[1:] {
			v, ok := coerce.ToUint(val)
			if !ok {
				return r.zctx.WrapError(ectx.Arena(), r.name+": not a number", val)
			}
			result = r.fn.Uint64(result, v)
		}
		return zed.NewUint64(result)
	case zed.IsSigned(id):
		result := val0.Int()
		for _, val := range args[1:] {
			//XXX this is really bad because we silently coerce
			// floats to ints if we hit a float first
			v, ok := coerce.ToInt(val)
			if !ok {
				return r.zctx.WrapError(ectx.Arena(), r.name+": not a number", val)
			}
			result = r.fn.Int64(result, v)
		}
		return zed.NewInt64(result)
	case zed.IsFloat(id):
		//XXX this is wrong like math aggregators...
		// need to be more robust and adjust type as new types encountered
		result := val0.Float()
		for _, val := range args[1:] {
			v, ok := coerce.ToFloat(val)
			if !ok {
				return r.zctx.WrapError(ectx.Arena(), r.name+": not a number", val)
			}
			result = r.fn.Float64(result, v)
		}
		return zed.NewFloat64(result)
	}
	return r.zctx.WrapError(ectx.Arena(), r.name+": not a number", val0)
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#round
type Round struct {
	zctx *zed.Context
}

func (r *Round) Call(ectx expr.Context, args []zed.Value) zed.Value {
	val := args[0]
	switch id := val.Type().ID(); {
	case zed.IsUnsigned(id) || zed.IsSigned(id):
		return val
	case zed.IsFloat(id):
		return zed.NewFloat(val.Type(), math.Round(val.Float()))
	}
	return r.zctx.WrapError(ectx.Arena(), "round: not a number", val)
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#pow
type Pow struct {
	zctx *zed.Context
}

func (p *Pow) Call(ectx expr.Context, args []zed.Value) zed.Value {
	x, ok := coerce.ToFloat(args[0])
	if !ok {
		return p.zctx.WrapError(ectx.Arena(), "pow: not a number", args[0])
	}
	y, ok := coerce.ToFloat(args[1])
	if !ok {
		return p.zctx.WrapError(ectx.Arena(), "pow: not a number", args[1])
	}
	return zed.NewFloat64(math.Pow(x, y))
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#sqrt
type Sqrt struct {
	zctx *zed.Context
}

func (s *Sqrt) Call(ectx expr.Context, args []zed.Value) zed.Value {
	x, ok := coerce.ToFloat(args[0])
	if !ok {
		return s.zctx.WrapError(ectx.Arena(), "sqrt: not a number", args[0])
	}
	return zed.NewFloat64(math.Sqrt(x))
}

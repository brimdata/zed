package function

import (
	"errors"
	"fmt"
	"math"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/anymath"
	"github.com/brimdata/zed/expr/coerce"
	"github.com/brimdata/zed/expr/result"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#abs.md
type Abs struct {
	stash result.Value
}

func (a *Abs) Call(args []zed.Value) *zed.Value {
	v := args[0]
	id := v.Type.ID()
	if zed.IsFloat(id) {
		f, err := zed.DecodeFloat64(v.Bytes)
		if err != nil {
			panic(fmt.Errorf("abs: corrupt Zed bytes: %w", err))
		}
		f = math.Abs(f)
		return a.stash.Float64(f)
	}
	if !zed.IsInteger(id) {
		return a.stash.Error(errors.New("abs: not a number"))
	}
	if !zed.IsSigned(id) {
		return a.stash.Copy(&args[0])
	}
	x, err := zed.DecodeInt(v.Bytes)
	if err != nil {
		panic(fmt.Errorf("abs: corrupt Zed bytes: %w", err))
	}
	if x < 0 {
		x = -x
	}
	return a.stash.Int64(x)
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#ceil
type Ceil struct {
	stash result.Value
}

func (c *Ceil) Call(args []zed.Value) *zed.Value {
	v := args[0]
	id := v.Type.ID()
	switch {
	case zed.IsFloat(id):
		f, err := zed.DecodeFloat64(v.Bytes)
		if err != nil {
			panic(fmt.Errorf("floor: corrupt Zed bytes: %w", err))
		}
		f = math.Ceil(f)
		return c.stash.Float64(f)
	case zed.IsInteger(id):
		return c.stash.Copy(&args[0])
	default:
		return c.stash.Error(errors.New("ceil: not a number"))
	}
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#floor
type Floor struct {
	stash result.Value
}

func (f *Floor) Call(args []zed.Value) *zed.Value {
	v := args[0]
	id := v.Type.ID()
	switch {
	case zed.IsFloat(id):
		v, _ := zed.DecodeFloat64(v.Bytes)
		v = math.Floor(v)
		return f.stash.Float64(v)
	case zed.IsInteger(id):
		return f.stash.Copy(&args[0])
	default:
		return f.stash.Error(errors.New("floor: not a number"))
	}
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#log
type Log struct {
	stash result.Value
}

func (l *Log) Call(args []zed.Value) *zed.Value {
	x, ok := coerce.ToFloat(args[0])
	if !ok {
		return l.stash.Error(errors.New("log: numeric argument required"))
	}
	if x <= 0 {
		return l.stash.Error(errors.New("log: negative argument"))
	}
	return l.stash.Float64(math.Log(x))
}

type reducer struct {
	stash result.Value
	fn    *anymath.Function
}

func (r *reducer) Call(args []zed.Value) *zed.Value {
	zv := args[0]
	typ := zv.Type
	id := typ.ID()
	if zed.IsFloat(id) {
		//XXX this is wrong like math aggregators...
		// need to be more robust and adjust type as new types encountered
		result, _ := zed.DecodeFloat64(zv.Bytes)
		for _, zv := range args[1:] {
			v, ok := coerce.ToFloat(zv)
			if !ok {
				return r.stash.Error(errors.New("not a number"))
			}
			result = r.fn.Float64(result, v)
		}
		return r.stash.Float64(result)
	}
	if !zed.IsNumber(id) {
		return r.stash.Error(errors.New("not a number"))
	}
	if zed.IsSigned(id) {
		result, _ := zed.DecodeInt(zv.Bytes)
		for _, zv := range args[1:] {
			//XXX this is really bad because we silently coerce
			// floats to ints if we hit a float first
			v, ok := coerce.ToInt(zv)
			if !ok {
				return r.stash.Error(errors.New("not a number"))
			}
			result = r.fn.Int64(result, v)
		}
		return r.stash.Int64(result)
	}
	result, _ := zed.DecodeUint(zv.Bytes)
	for _, zv := range args[1:] {
		v, ok := coerce.ToUint(zv)
		if !ok {
			//XXX this is bad
			return r.stash.Error(errors.New("not a number"))
		}
		result = r.fn.Uint64(result, v)
	}
	return r.stash.Uint64(result)
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#round
type Round struct {
	stash result.Value
}

func (r *Round) Call(args []zed.Value) *zed.Value {
	zv := args[0]
	id := zv.Type.ID()
	if zed.IsFloat(id) {
		f, err := zed.DecodeFloat64(zv.Bytes)
		if err != nil {
			panic(fmt.Errorf("round: corrupt Zed bytes: %w", err))
		}
		return r.stash.Float64(math.Round(f))
	}
	if !zed.IsNumber(id) {
		return r.stash.Error(errors.New("round: not a number"))
	}
	return r.stash.Copy(&args[0])
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#pow
type Pow struct {
	stash result.Value
}

func (p *Pow) Call(args []zed.Value) *zed.Value {
	x, ok := coerce.ToFloat(args[0])
	if !ok {
		return p.stash.Error(errors.New("pow: not a number"))
	}
	y, ok := coerce.ToFloat(args[1])
	if !ok {
		return p.stash.Error(errors.New("pow: not a number"))
	}
	r := math.Pow(x, y)
	//XXX shouldn't we just let IEEE NaN through?
	if math.IsNaN(r) {
		return p.stash.Error(errors.New("pow: NaN"))
	}
	return p.stash.Float64(r)
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#sqrt
type Sqrt struct {
	stash result.Value
}

func (s *Sqrt) Call(args []zed.Value) *zed.Value {
	x, ok := coerce.ToFloat(args[0])
	if !ok {
		return s.stash.Error(errors.New("sqrt: not a number"))
	}
	x = math.Sqrt(x)
	//XXX let NaN through
	if math.IsNaN(x) {
		return s.stash.Error(errors.New("sqrt: not a number"))
	}
	return s.stash.Float64(x)
}

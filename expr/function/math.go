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

//XXX rework result.Buffer

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
			panic(fmt.Errorf("abs: corrupt Zed bytes", err))
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
		panic(fmt.Errorf("abs: corrupt Zed bytes", err))
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
			panic(fmt.Errorf("floor: corrupt Zed bytes", err))
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
	result.Buffer
	fn *anymath.Function
}

func (r *reducer) Call(args []zed.Value) *zed.Value {
	zv := args[0]
	typ := zv.Type
	id := typ.ID()
	if zed.IsFloat(id) {
		result, _ := zed.DecodeFloat64(zv.Bytes)
		for _, zv := range args[1:] {
			v, ok := coerce.ToFloat(zv)
			if !ok {
				return zed.Value{}, ErrBadArgument
			}
			result = r.fn.Float64(result, v)
		}
		return zed.Value{typ, r.Float64(result)}, nil
	}
	if !zed.IsNumber(id) {
		// XXX better message
		return zed.Value{}, ErrBadArgument
	}
	if zed.IsSigned(id) {
		result, _ := zed.DecodeInt(zv.Bytes)
		for _, zv := range args[1:] {
			v, ok := coerce.ToInt(zv)
			if !ok {
				// XXX better message
				return zed.Value{}, ErrBadArgument
			}
			result = r.fn.Int64(result, v)
		}
		return zed.Value{typ, r.Int(result)}, nil
	}
	result, _ := zed.DecodeUint(zv.Bytes)
	for _, zv := range args[1:] {
		v, ok := coerce.ToUint(zv)
		if !ok {
			// XXX better message
			return zed.Value{}, ErrBadArgument
		}
		result = r.fn.Uint64(result, v)
	}
	return zed.Value{typ, r.Uint(result)}, nil
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#round
type Round struct {
	result.Buffer
}

func (r *Round) Call(args []zed.Value) *zed.Value {
	zv := args[0]
	id := zv.Type.ID()
	if zed.IsFloat(id) {
		f, _ := zed.DecodeFloat64(zv.Bytes)
		return zed.Value{zv.Type, r.Float64(math.Round(f))}, nil

	}
	if !zed.IsNumber(id) {
		return badarg("round")
	}
	return zv, nil
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#pow
type Pow struct {
	result.Buffer
}

func (p *Pow) Call(args []zed.Value) *zed.Value {
	x, ok := coerce.ToFloat(args[0])
	if !ok {
		return badarg("pow")
	}
	y, ok := coerce.ToFloat(args[1])
	if !ok {
		return badarg("pow")
	}
	r := math.Pow(x, y)
	if math.IsNaN(r) {
		return badarg("pow")
	}
	return zed.Value{zed.TypeFloat64, p.Float64(r)}, nil
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#sqrt
type Sqrt struct {
	result.Buffer
}

func (s *Sqrt) Call(args []zed.Value) *zed.Value {
	x, ok := coerce.ToFloat(args[0])
	if !ok {
		return badarg("sqrt")
	}
	x = math.Sqrt(x)
	if math.IsNaN(x) {
		// For now we can't represent non-numeric values in a float64,
		// we will revisit this but it has implications for file
		// formats, the Zed language, etc.
		return badarg("sqrt")
	}
	return zed.Value{zed.TypeFloat64, s.Float64(x)}, nil
}

package function

import (
	"math"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/anymath"
	"github.com/brimdata/zed/expr/coerce"
	"github.com/brimdata/zed/expr/result"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#abs.md
type Abs struct {
	result.Buffer
}

func (a *Abs) Call(args []zed.Value) (zed.Value, error) {
	v := args[0]
	id := v.Type.ID()
	if zed.IsFloat(id) {
		f, _ := zed.DecodeFloat64(v.Bytes)
		f = math.Abs(f)
		return zed.Value{zed.TypeFloat64, a.Float64(f)}, nil
	}
	if !zed.IsInteger(id) {
		return badarg("abs")
	}
	if !zed.IsSigned(id) {
		return v, nil
	}
	x, _ := zed.DecodeInt(v.Bytes)
	if x < 0 {
		x = -x
	}
	return zed.Value{v.Type, a.Int(x)}, nil
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#ceil
type Ceil struct {
	result.Buffer
}

func (c *Ceil) Call(args []zed.Value) (zed.Value, error) {
	v := args[0]
	id := v.Type.ID()
	if zed.IsFloat(id) {
		f, _ := zed.DecodeFloat64(v.Bytes)
		f = math.Ceil(f)
		return zed.Value{zed.TypeFloat64, c.Float64(f)}, nil
	}
	if zed.IsInteger(id) {
		return v, nil
	}
	return badarg("ceil")
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#floor
type Floor struct {
	result.Buffer
}

func (f *Floor) Call(args []zed.Value) (zed.Value, error) {
	v := args[0]
	id := v.Type.ID()
	if zed.IsFloat(id) {
		v, _ := zed.DecodeFloat64(v.Bytes)
		v = math.Floor(v)
		return zed.Value{zed.TypeFloat64, f.Float64(v)}, nil
	}
	if zed.IsInteger(id) {
		return v, nil
	}
	return badarg("floor")
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#log
type Log struct {
	result.Buffer
}

func (l *Log) Call(args []zed.Value) (zed.Value, error) {
	x, ok := coerce.ToFloat(args[0])
	// XXX should have better error messages
	if !ok {
		return badarg("log")
	}
	if x <= 0 {
		return badarg("log")
	}
	return zed.Value{zed.TypeFloat64, l.Float64(math.Log(x))}, nil
}

type reducer struct {
	result.Buffer
	fn *anymath.Function
}

func (r *reducer) Call(args []zed.Value) (zed.Value, error) {
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

func (r *Round) Call(args []zed.Value) (zed.Value, error) {
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

func (p *Pow) Call(args []zed.Value) (zed.Value, error) {
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

func (s *Sqrt) Call(args []zed.Value) (zed.Value, error) {
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

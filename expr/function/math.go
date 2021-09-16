package function

import (
	"math"

	"github.com/brimdata/zed/anymath"
	"github.com/brimdata/zed/expr/coerce"
	"github.com/brimdata/zed/expr/result"
	"github.com/brimdata/zed/zng"
)

type abs struct {
	result.Buffer
}

func (a *abs) Call(args []zng.Value) (zng.Value, error) {
	v := args[0]
	id := v.Type.ID()
	if zng.IsFloat(id) {
		f, _ := zng.DecodeFloat64(v.Bytes)
		f = math.Abs(f)
		return zng.Value{zng.TypeFloat64, a.Float64(f)}, nil
	}
	if !zng.IsInteger(id) {
		return badarg("abs")
	}
	if !zng.IsSigned(id) {
		return v, nil
	}
	x, _ := zng.DecodeInt(v.Bytes)
	if x < 0 {
		x = -x
	}
	return zng.Value{v.Type, a.Int(x)}, nil
}

type ceil struct {
	result.Buffer
}

func (c *ceil) Call(args []zng.Value) (zng.Value, error) {
	v := args[0]
	id := v.Type.ID()
	if zng.IsFloat(id) {
		f, _ := zng.DecodeFloat64(v.Bytes)
		f = math.Ceil(f)
		return zng.Value{zng.TypeFloat64, c.Float64(f)}, nil
	}
	if zng.IsInteger(id) {
		return v, nil
	}
	return badarg("ceil")
}

type floor struct {
	result.Buffer
}

func (f *floor) Call(args []zng.Value) (zng.Value, error) {
	v := args[0]
	id := v.Type.ID()
	if zng.IsFloat(id) {
		v, _ := zng.DecodeFloat64(v.Bytes)
		v = math.Floor(v)
		return zng.Value{zng.TypeFloat64, f.Float64(v)}, nil
	}
	if zng.IsInteger(id) {
		return v, nil
	}
	return badarg("floor")
}

type log struct {
	result.Buffer
}

func (l *log) Call(args []zng.Value) (zng.Value, error) {
	x, ok := coerce.ToFloat(args[0])
	// XXX should have better error messages
	if !ok {
		return badarg("log")
	}
	if x <= 0 {
		return badarg("log")
	}
	return zng.Value{zng.TypeFloat64, l.Float64(math.Log(x))}, nil
}

type reducer struct {
	result.Buffer
	fn *anymath.Function
}

func (r *reducer) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	typ := zv.Type
	id := typ.ID()
	if zng.IsFloat(id) {
		result, _ := zng.DecodeFloat64(zv.Bytes)
		for _, zv := range args[1:] {
			v, ok := coerce.ToFloat(zv)
			if !ok {
				return zng.Value{}, ErrBadArgument
			}
			result = r.fn.Float64(result, v)
		}
		return zng.Value{typ, r.Float64(result)}, nil
	}
	if !zng.IsNumber(id) {
		// XXX better message
		return zng.Value{}, ErrBadArgument
	}
	if zng.IsSigned(id) {
		result, _ := zng.DecodeInt(zv.Bytes)
		for _, zv := range args[1:] {
			v, ok := coerce.ToInt(zv)
			if !ok {
				// XXX better message
				return zng.Value{}, ErrBadArgument
			}
			result = r.fn.Int64(result, v)
		}
		return zng.Value{typ, r.Int(result)}, nil
	}
	result, _ := zng.DecodeUint(zv.Bytes)
	for _, zv := range args[1:] {
		v, ok := coerce.ToUint(zv)
		if !ok {
			// XXX better message
			return zng.Value{}, ErrBadArgument
		}
		result = r.fn.Uint64(result, v)
	}
	return zng.Value{typ, r.Uint(result)}, nil
}

type mod struct {
	result.Buffer
}

//XXX currently integer mod, but this could also do fmod
// also why doesn't Zed have x%y instead of mod(x,y)?
func (m *mod) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	id := zv.Type.ID()
	if zng.IsFloat(id) {
		return badarg("mod")
	}
	y, ok := coerce.ToUint(args[1])
	if !ok {
		return badarg("mod")
	}
	if !zng.IsNumber(id) {
		return badarg("mod")
	}
	if zng.IsSigned(id) {
		x, _ := zng.DecodeInt(zv.Bytes)
		return zng.Value{zv.Type, m.Int(x % int64(y))}, nil
	}
	x, _ := zng.DecodeUint(zv.Bytes)
	return zng.Value{zv.Type, m.Uint(x % y)}, nil
}

type round struct {
	result.Buffer
}

func (r *round) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	id := zv.Type.ID()
	if zng.IsFloat(id) {
		f, _ := zng.DecodeFloat64(zv.Bytes)
		return zng.Value{zv.Type, r.Float64(math.Round(f))}, nil

	}
	if !zng.IsNumber(id) {
		return badarg("round")
	}
	return zv, nil
}

type pow struct {
	result.Buffer
}

func (p *pow) Call(args []zng.Value) (zng.Value, error) {
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
	return zng.Value{zng.TypeFloat64, p.Float64(r)}, nil
}

type sqrt struct {
	result.Buffer
}

func (s *sqrt) Call(args []zng.Value) (zng.Value, error) {
	x, ok := coerce.ToFloat(args[0])
	if !ok {
		return badarg("sqrt")
	}
	x = math.Sqrt(x)
	if math.IsNaN(x) {
		// For now we can't represent non-numeric values in a float64,
		// we will revisit this but it has implications for file
		// formats, Zed, etc.
		return badarg("sqrt")
	}
	return zng.Value{zng.TypeFloat64, s.Float64(x)}, nil
}

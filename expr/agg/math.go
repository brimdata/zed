package agg

import (
	"github.com/brimsec/zq/anymath"
	"github.com/brimsec/zq/expr/coerce"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type consumer interface {
	result() zng.Value
	consume(zng.Value) error
}

type mathReducer struct {
	function *anymath.Function
	typ      zng.Type
	math     consumer
}

func newMathReducer(f *anymath.Function) *mathReducer {
	return &mathReducer{function: f}
}

func (m *mathReducer) Result(*resolver.Context) (zng.Value, error) {
	if m.math == nil {
		if m.typ == nil {
			return zng.Value{Type: zng.TypeNull, Bytes: nil}, nil
		}
		return zng.Value{Type: m.typ, Bytes: nil}, nil
	}
	return m.math.result(), nil
}

func (m *mathReducer) Consume(v zng.Value) error {
	if v.Type == nil {
		//m.FieldNotFound++
		return nil
	}
	m.consumeVal(v)
	return nil
}

func (m *mathReducer) consumeVal(val zng.Value) {
	// A numerical reducer inherits the type of the first numeric
	// value it sees and coerces all future instances of this value
	// to this initial type.
	if m.typ == nil {
		m.typ = val.Type
	}
	if val.Bytes == nil {
		return
	}
	if m.math == nil {
		switch val.Type.ID() {
		case zng.IdInt8, zng.IdInt16, zng.IdInt32, zng.IdInt64:
			m.math = NewInt64(m.function)
		case zng.IdUint8, zng.IdUint16, zng.IdUint32, zng.IdUint64:
			m.math = NewUint64(m.function)
		case zng.IdFloat64:
			m.math = NewFloat64(m.function)
		case zng.IdDuration:
			m.math = NewDuration(m.function)
		case zng.IdTime:
			m.math = NewTime(m.function)
		default:
			//m.TypeMismatch++
			return
		}
	}
	if m.math.consume(val) == zng.ErrTypeMismatch {
		//m.TypeMismatch++
	}
}

func (m *mathReducer) ResultAsPartial(*resolver.Context) (zng.Value, error) {
	return m.Result(nil)
}

func (m *mathReducer) ConsumeAsPartial(v zng.Value) error {
	m.consumeVal(v)
	return nil
}

type Float64 struct {
	state    float64
	function anymath.Float64
}

func NewFloat64(f *anymath.Function) *Float64 {
	return &Float64{
		state:    f.Init.Float64,
		function: f.Float64,
	}
}

func (f *Float64) result() zng.Value {
	return zng.NewFloat64(f.state)
}

func (f *Float64) consume(v zng.Value) error {
	if v, ok := coerce.ToFloat(v); ok {
		f.state = f.function(f.state, v)
		return nil
	}
	return zng.ErrTypeMismatch
}

type Int64 struct {
	state    int64
	function anymath.Int64
}

func NewInt64(f *anymath.Function) *Int64 {
	return &Int64{
		state:    f.Init.Int64,
		function: f.Int64,
	}
}

func (i *Int64) result() zng.Value {
	return zng.Value{zng.TypeInt64, zng.EncodeInt(i.state)}
}

func (i *Int64) consume(v zng.Value) error {
	if v, ok := coerce.ToInt(v); ok {
		i.state = i.function(i.state, v)
		return nil
	}
	return zng.ErrTypeMismatch
}

type Uint64 struct {
	state    uint64
	function anymath.Uint64
}

func NewUint64(f *anymath.Function) *Uint64 {
	return &Uint64{
		state:    f.Init.Uint64,
		function: f.Uint64,
	}
}

func (u *Uint64) result() zng.Value {
	return zng.Value{zng.TypeUint64, zng.EncodeUint(u.state)}
}

func (u *Uint64) consume(v zng.Value) error {
	if v, ok := coerce.ToUint(v); ok {
		u.state = u.function(u.state, v)
		return nil
	}
	return zng.ErrTypeMismatch
}

type Duration struct {
	state    int64
	function anymath.Int64
}

func NewDuration(f *anymath.Function) *Duration {
	return &Duration{
		state:    f.Init.Int64,
		function: f.Int64,
	}
}

func (d *Duration) result() zng.Value {
	return zng.NewDuration(d.state)
}

func (d *Duration) consume(v zng.Value) error {
	if v, ok := coerce.ToDuration(v); ok {
		d.state = d.function(d.state, v)
		return nil
	}
	return zng.ErrTypeMismatch
}

type Time struct {
	state    nano.Ts
	function anymath.Int64
}

func NewTime(f *anymath.Function) *Time {
	return &Time{
		state:    nano.Ts(f.Init.Int64),
		function: f.Int64,
	}
}

func (t *Time) result() zng.Value {
	return zng.NewTime(t.state)
}

func (t *Time) consume(v zng.Value) error {
	if v, ok := coerce.ToTime(v); ok {
		t.state = nano.Ts(t.function(int64(t.state), int64(v)))
		return nil
	}
	return zng.ErrTypeMismatch
}

package agg

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/anymath"
	"github.com/brimdata/zed/expr/coerce"
	"github.com/brimdata/zed/pkg/nano"
)

type consumer interface {
	result() zed.Value
	consume(zed.Value) error
}

type mathReducer struct {
	function *anymath.Function
	typ      zed.Type
	math     consumer
}

var _ Function = (*mathReducer)(nil)

func newMathReducer(f *anymath.Function) *mathReducer {
	return &mathReducer{function: f}
}

func (m *mathReducer) Result(*zed.Context) zed.Value {
	if m.math == nil {
		if m.typ == nil {
			return zed.Null
		}
		return zed.Value{Type: m.typ}
	}
	return m.math.result()
}

func (m *mathReducer) Consume(v zed.Value) {
	//XXX why would type be nil here?
	if v.Type != nil {
		m.consumeVal(v)
	}
}

func (m *mathReducer) consumeVal(val zed.Value) {
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
		case zed.IDInt8, zed.IDInt16, zed.IDInt32, zed.IDInt64:
			m.math = NewInt64(m.function)
		case zed.IDUint8, zed.IDUint16, zed.IDUint32, zed.IDUint64:
			m.math = NewUint64(m.function)
		case zed.IDFloat32, zed.IDFloat64:
			m.math = NewFloat64(m.function)
		case zed.IDDuration:
			m.math = NewDuration(m.function)
		case zed.IDTime:
			m.math = NewTime(m.function)
		default:
			//m.TypeMismatch++
			return
		}
	}
	if m.math.consume(val) == zed.ErrTypeMismatch {
		//m.TypeMismatch++
	}
}

func (m *mathReducer) ResultAsPartial(*zed.Context) zed.Value {
	return m.Result(nil)
}

func (m *mathReducer) ConsumeAsPartial(v zed.Value) {
	m.consumeVal(v)
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

func (f *Float64) result() zed.Value {
	return zed.NewFloat64(f.state)
}

func (f *Float64) consume(v zed.Value) error {
	if v, ok := coerce.ToFloat(v); ok {
		f.state = f.function(f.state, v)
		return nil
	}
	return zed.ErrTypeMismatch
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

func (i *Int64) result() zed.Value {
	return zed.Value{zed.TypeInt64, zed.EncodeInt(i.state)}
}

func (i *Int64) consume(v zed.Value) error {
	if v, ok := coerce.ToInt(v); ok {
		i.state = i.function(i.state, v)
		return nil
	}
	return zed.ErrTypeMismatch
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

func (u *Uint64) result() zed.Value {
	return zed.Value{zed.TypeUint64, zed.EncodeUint(u.state)}
}

func (u *Uint64) consume(v zed.Value) error {
	if v, ok := coerce.ToUint(v); ok {
		u.state = u.function(u.state, v)
		return nil
	}
	return zed.ErrTypeMismatch
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

func (d *Duration) result() zed.Value {
	return zed.NewDuration(nano.Duration(d.state))
}

func (d *Duration) consume(v zed.Value) error {
	if v, ok := coerce.ToDuration(v); ok {
		d.state = d.function(d.state, int64(v))
		return nil
	}
	return zed.ErrTypeMismatch
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

func (t *Time) result() zed.Value {
	return zed.NewTime(t.state)
}

func (t *Time) consume(v zed.Value) error {
	if v, ok := coerce.ToTime(v); ok {
		t.state = nano.Ts(t.function(int64(t.state), int64(v)))
		return nil
	}
	return zed.ErrTypeMismatch
}

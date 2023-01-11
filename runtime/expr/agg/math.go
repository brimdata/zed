package agg

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/anymath"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/runtime/expr/coerce"
	"github.com/brimdata/zed/zson"
)

type consumer interface {
	result() *zed.Value
	consume(*zed.Value)
	typ() zed.Type
}

type mathReducer struct {
	function *anymath.Function
	hasval   bool
	math     consumer
	pair     coerce.Pair
}

var _ Function = (*mathReducer)(nil)

func newMathReducer(f *anymath.Function) *mathReducer {
	return &mathReducer{function: f}
}

func (m *mathReducer) Result(zctx *zed.Context) *zed.Value {
	if !m.hasval {
		if m.math == nil {
			return zed.Null
		}
		return zed.NewValue(m.math.typ(), nil)
	}
	return m.math.result()
}

func (m *mathReducer) Consume(val *zed.Value) {
	m.consumeVal(val)
}

func (m *mathReducer) consumeVal(val *zed.Value) {
	var id int
	if m.math != nil {
		var err error
		// XXX We're not using the value coercion parts of coerce.Pair here.
		// Would be better if coerce had a function that just compared types
		// and returned the type to coerce to.
		id, err = m.pair.Coerce(zed.NewValue(m.math.typ(), nil), val)
		if err != nil {
			// Skip invalid values.
			return
		}
	} else {
		id = val.Type.ID()
	}
	if m.math == nil || m.math.typ().ID() != id {
		state := zed.Null
		if m.math != nil {
			state = m.math.result()
		}
		switch id {
		case zed.IDInt8, zed.IDInt16, zed.IDInt32, zed.IDInt64:
			m.math = NewInt64(m.function, state)
		case zed.IDUint8, zed.IDUint16, zed.IDUint32, zed.IDUint64:
			m.math = NewUint64(m.function, state)
		case zed.IDFloat16, zed.IDFloat32, zed.IDFloat64:
			m.math = NewFloat64(m.function, state)
		case zed.IDDuration:
			m.math = NewDuration(m.function, state)
		case zed.IDTime:
			m.math = NewTime(m.function, state)
		default:
			// Ignore types we can't handle.
			return
		}
	}
	if val.IsNull() {
		return
	}
	m.hasval = true
	m.math.consume(val)
}

func (m *mathReducer) ResultAsPartial(*zed.Context) *zed.Value {
	return m.Result(nil)
}

func (m *mathReducer) ConsumeAsPartial(val *zed.Value) {
	m.consumeVal(val)
}

type Float64 struct {
	state    float64
	function anymath.Float64
}

func NewFloat64(f *anymath.Function, val *zed.Value) *Float64 {
	state := f.Init.Float64
	if val.Bytes != nil {
		var ok bool
		state, ok = coerce.ToFloat(val)
		if !ok {
			panicCoercionFail(zed.TypeFloat64, val.Type)
		}
	}
	return &Float64{
		state:    state,
		function: f.Float64,
	}
}

func (f *Float64) result() *zed.Value {
	return zed.NewValue(zed.TypeFloat64, zed.EncodeFloat64(f.state))
}

func (f *Float64) consume(val *zed.Value) {
	if v, ok := coerce.ToFloat(val); ok {
		f.state = f.function(f.state, v)
	}
}

func (f *Float64) typ() zed.Type { return zed.TypeFloat64 }

type Int64 struct {
	state    int64
	function anymath.Int64
}

func NewInt64(f *anymath.Function, val *zed.Value) *Int64 {
	state := f.Init.Int64
	if !val.IsNull() {
		var ok bool
		state, ok = coerce.ToInt(val)
		if !ok {
			panicCoercionFail(zed.TypeInt64, val.Type)
		}
	}
	return &Int64{
		state:    state,
		function: f.Int64,
	}
}

func (i *Int64) result() *zed.Value {
	return zed.NewValue(zed.TypeInt64, zed.EncodeInt(i.state))
}

func (i *Int64) consume(val *zed.Value) {
	if v, ok := coerce.ToInt(val); ok {
		i.state = i.function(i.state, v)
	}
}

func (f *Int64) typ() zed.Type { return zed.TypeInt64 }

type Uint64 struct {
	state    uint64
	function anymath.Uint64
}

func NewUint64(f *anymath.Function, val *zed.Value) *Uint64 {
	state := f.Init.Uint64
	if !val.IsNull() {
		var ok bool
		state, ok = coerce.ToUint(val)
		if !ok {
			panicCoercionFail(zed.TypeUint64, val.Type)
		}
	}
	return &Uint64{
		state:    state,
		function: f.Uint64,
	}
}

func (u *Uint64) result() *zed.Value {
	return zed.NewValue(zed.TypeUint64, zed.EncodeUint(u.state))
}

func (u *Uint64) consume(val *zed.Value) {
	if v, ok := coerce.ToUint(val); ok {
		u.state = u.function(u.state, v)
	}
}

func (f *Uint64) typ() zed.Type { return zed.TypeUint64 }

type Duration struct {
	state    int64
	function anymath.Int64
}

func NewDuration(f *anymath.Function, val *zed.Value) *Duration {
	state := f.Init.Int64
	if !val.IsNull() {
		var ok bool
		state, ok = coerce.ToInt(val)
		if !ok {
			panicCoercionFail(zed.TypeDuration, val.Type)
		}
	}
	return &Duration{
		state:    state,
		function: f.Int64,
	}
}

func (d *Duration) result() *zed.Value {
	return zed.NewValue(zed.TypeDuration, zed.EncodeDuration(nano.Duration(d.state)))
}

func (d *Duration) consume(val *zed.Value) {
	if v, ok := coerce.ToDuration(val); ok {
		d.state = d.function(d.state, int64(v))
	}
}

func (f *Duration) typ() zed.Type { return zed.TypeDuration }

type Time struct {
	state    nano.Ts
	function anymath.Int64
}

func NewTime(f *anymath.Function, val *zed.Value) *Time {
	state := f.Init.Int64
	if !val.IsNull() {
		var ok bool
		state, ok = coerce.ToInt(val)
		if !ok {
			panicCoercionFail(zed.TypeTime, val.Type)
		}
	}
	return &Time{
		state:    nano.Ts(state),
		function: f.Int64,
	}
}

func (t *Time) result() *zed.Value {
	return zed.NewValue(zed.TypeTime, zed.EncodeTime(t.state))
}

func (t *Time) consume(val *zed.Value) {
	if v, ok := coerce.ToTime(val); ok {
		t.state = nano.Ts(t.function(int64(t.state), int64(v)))
	}
}

func (f *Time) typ() zed.Type { return zed.TypeTime }

func panicCoercionFail(to, from zed.Type) {
	panic(fmt.Sprintf("internal aggregation error: cannot coerce %s to %s", zson.String(from), zson.String(to)))
}

package field

import (
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/reducer"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type Streamfn interface {
	Result() zng.Value
	Consume(zng.Value) error
}

type FieldReducer struct {
	reducer.Reducer
	Op       string
	Resolver *expr.FieldExpr
	typ      zng.Type
	fn       Streamfn
}

func (fr *FieldReducer) Result() zng.Value {
	if fr.fn == nil {
		if fr.typ == nil {
			return zng.Value{Type: zng.TypeNull, Bytes: nil}
		}
		return zng.Value{Type: fr.typ, Bytes: nil}
	}
	return fr.fn.Result()
}

func (fr *FieldReducer) Consume(r *zng.Record) {
	// XXX for now, we create a new zng.Value everytime we operate on
	// a field.  this could be made more efficient by having each typed
	// reducer just parse the byte slice in the record without making a value...
	// XXX then we have Values in the zbuf.Record, we would first check the
	// Value element in the column--- this would all go in a new method of zbuf.Record
	val, err := fr.Resolver.Eval(r)
	if err != nil || val.Type == nil {
		fr.FieldNotFound++
		return
	}
	fr.consumeVal(val)
}

func (fr *FieldReducer) consumeVal(val zng.Value) {
	if fr.typ == nil {
		fr.typ = val.Type
	}
	if val.Bytes == nil {
		return
	}
	if fr.fn == nil {
		switch val.Type.ID() {
		case zng.IdInt16, zng.IdInt32, zng.IdInt64:
			fr.fn = NewIntStreamfn(fr.Op)
		case zng.IdUint16, zng.IdUint32, zng.IdUint64:
			fr.fn = NewUintStreamfn(fr.Op)
		case zng.IdFloat64:
			fr.fn = NewFloat64Streamfn(fr.Op)
		case zng.IdDuration:
			fr.fn = NewDurationStreamfn(fr.Op)
		case zng.IdTime:
			fr.fn = NewTimeStreamfn(fr.Op)
		default:
			fr.TypeMismatch++
			return
		}
	}
	if fr.fn.Consume(val) == zng.ErrTypeMismatch {
		fr.TypeMismatch++
	}
}

func (fr *FieldReducer) ResultPart(*resolver.Context) (zng.Value, error) {
	return fr.Result(), nil
}

func (fr *FieldReducer) ConsumePart(v zng.Value) error {
	fr.consumeVal(v)
	return nil
}

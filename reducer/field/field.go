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

type FieldProto struct {
	target              string
	op                  string
	resolver, tresolver expr.FieldExprResolver
}

func NewFieldProto(target string, tresolver, resolver expr.FieldExprResolver, op string) *FieldProto {
	return &FieldProto{
		target:    target,
		tresolver: tresolver,
		resolver:  resolver,
		op:        op,
	}
}

func (fp *FieldProto) Target() string {
	return fp.target
}

func (fp *FieldProto) TargetResolver() expr.FieldExprResolver {
	return fp.tresolver
}

func (fp *FieldProto) Instantiate() reducer.Interface {
	return &FieldReducer{op: fp.op, resolver: fp.resolver}
}

type FieldReducer struct {
	reducer.Reducer
	op       string
	resolver expr.FieldExprResolver
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
	val := fr.resolver(r)
	if val.Type == nil {
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
			fr.fn = NewIntStreamfn(fr.op)
		case zng.IdUint16, zng.IdUint32, zng.IdUint64:
			fr.fn = NewUintStreamfn(fr.op)
		case zng.IdFloat64:
			fr.fn = NewFloat64Streamfn(fr.op)
		case zng.IdDuration:
			fr.fn = NewDurationStreamfn(fr.op)
		case zng.IdTime:
			fr.fn = NewTimeStreamfn(fr.op)
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

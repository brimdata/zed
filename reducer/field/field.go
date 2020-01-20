package field

import (
	"fmt"
	"github.com/mccanne/zq/reducer"
	"github.com/mccanne/zq/zng"
)

type Streamfn interface {
	Result() zng.Value
	Consume(zng.Value) error
}

type FieldProto struct {
	target string
	op     string
	field  string
}

func (fp *FieldProto) Target() string {
	return fp.target
}

func (fp *FieldProto) Instantiate(recType *zng.TypeRecord) reducer.Interface {
	typ, ok := recType.TypeOfField(fp.field)
	if !ok {
		panic(fmt.Sprintf("instantiate %s(%s) on type without field %s", fp.op, fp.field, fp.field))
	}
	return &FieldReducer{op: fp.op, field: fp.field, typ: typ}
}

func NewFieldProto(target, field, op string) *FieldProto {
	return &FieldProto{target, op, field}
}

type FieldReducer struct {
	reducer.Reducer
	op    string
	field string
	typ   zng.Type
	fn    Streamfn
}

func (fr *FieldReducer) Result() zng.Value {
	if fr.fn == nil {
		return zng.Value{}
	}
	return fr.fn.Result()
}

func (fr *FieldReducer) Consume(r *zng.Record) {
	// XXX for now, we create a new zng.Value everytime we operate on
	// a field.  this could be made more efficient by having each typed
	// reducer just parse the byte slice in the record without making a value...
	// XXX then we have Values in the zbuf.Record, we would first check the
	// Value element in the column--- this would all go in a new method of zbuf.Record
	val, err := r.ValueByField(fr.field)
	if err != nil {
		fr.FieldNotFound++
		return
	}

	if fr.fn == nil {
		switch val.Type.(type) {
		case *zng.TypeOfInt:
			fr.fn = NewIntStreamfn(fr.op)
		case *zng.TypeOfCount:
			fr.fn = NewCountStreamfn(fr.op)
		case *zng.TypeOfDouble:
			fr.fn = NewDoubleStreamfn(fr.op)
		case *zng.TypeOfInterval:
			fr.fn = NewIntervalStreamfn(fr.op)
		case *zng.TypeOfTime:
			fr.fn = NewTimeStreamfn(fr.op)
		default:
			fr.TypeMismatch++
			return
		}
	}

	err = fr.fn.Consume(val)
	if err == zng.ErrTypeMismatch {
		fr.TypeMismatch++
	}
}

func (fr *FieldReducer) ResultType() zng.Type {
	return fr.typ
}

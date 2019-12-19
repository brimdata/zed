package field

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zng"
	"github.com/mccanne/zq/reducer"
)

type Streamfn interface {
	Result() zeek.Value
	Consume(zeek.Value) error
}

type FieldProto struct {
	target string
	op     string
	field  string
}

func (fp *FieldProto) Target() string {
	return fp.target
}

func (fp *FieldProto) Instantiate() reducer.Interface {
	return &FieldReducer{op: fp.op, field: fp.field}
}

func NewFieldProto(target, field, op string) *FieldProto {
	return &FieldProto{target, op, field}
}

type FieldReducer struct {
	reducer.Reducer
	op    string
	field string
	fn    Streamfn
}

func (fr *FieldReducer) Result() zeek.Value {
	if fr.fn == nil {
		return &zeek.Unset{}
	}
	return fr.fn.Result()
}

func (fr *FieldReducer) Consume(r *zng.Record) {
	// XXX for now, we create a new zeek.Value everytime we operate on
	// a field.  this could be made more efficient by having each typed
	// reducer just parse the byte slice in the record without making a value...
	// XXX then we have Values in the zng.Record, we would first check the
	// Value element in the column--- this would all go in a new method of zng.Record
	val := r.ValueByField(fr.field)
	if val == nil {
		fr.FieldNotFound++
		return
	}

	if fr.fn == nil {
		switch val.Type().(type) {
		case *zeek.TypeOfInt:
			fr.fn = NewIntStreamfn(fr.op)
		case *zeek.TypeOfCount:
			fr.fn = NewCountStreamfn(fr.op)
		case *zeek.TypeOfDouble:
			fr.fn = NewDoubleStreamfn(fr.op)
		case *zeek.TypeOfInterval:
			fr.fn = NewIntervalStreamfn(fr.op)
		case *zeek.TypeOfTime:
			fr.fn = NewTimeStreamfn(fr.op)
		default:
			fr.TypeMismatch++
			return
		}
	}

	err := fr.fn.Consume(val)
	if err == zng.ErrTypeMismatch {
		fr.TypeMismatch++
	}
}

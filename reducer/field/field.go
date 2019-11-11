package field

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/reducer"
)

type Field struct {
	reducer.Reducer
	field string
}

func NewField(name, field string) Field {
	return Field{
		Reducer: reducer.New(name),
		field:   field,
	}
}

func (f *Field) lookup(r *zson.Record) zeek.Value {
	// XXX for now, we create a new zeek.Value everytime we operate on
	// a field.  this could be made more efficient by having each typed
	// reducer just parse the byte slice in the record without making a value...
	// XXX then we have Values in the zson.Record, we would first check the
	// Value element in the column--- this would all go in a new method of zson.Record
	v := r.ValueByField(f.field)
	if v == nil {
		f.FieldNotFound++
	}
	return v
}

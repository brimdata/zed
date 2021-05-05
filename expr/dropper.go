package expr

import (
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zng/builder"
	"github.com/brimdata/zed/zson"
)

type dropper struct {
	typ       *zng.TypeRecord
	builder   *builder.ColumnBuilder
	fieldRefs []Evaluator
}

func (d *dropper) drop(in *zng.Record) (*zng.Record, error) {
	b := d.builder
	b.Reset()
	for _, e := range d.fieldRefs {
		val, err := e.Eval(in)
		if err != nil {
			return nil, err
		}
		b.Append(val.Bytes, val.IsContainer())
	}
	zv, err := b.Encode()
	if err != nil {
		return nil, err
	}
	return zng.NewRecord(d.typ, zv), nil
}

type Dropper struct {
	zctx      *zson.Context
	fields    field.List
	resolvers []Evaluator
	droppers  map[int]*dropper
}

func NewDropper(zctx *zson.Context, fields field.List) *Dropper {
	return &Dropper{
		zctx:     zctx,
		fields:   fields,
		droppers: make(map[int]*dropper),
	}
}

func (d *Dropper) newDropper(r *zng.Record) (*dropper, error) {
	fields, fieldTypes := complementFields(d.fields, nil, zng.TypeRecordOf(r.Type))
	// If the set of dropped fields is equal to the all of record's
	// fields, then there is no output for this input type.
	// We return nil to block this input type.
	if len(fieldTypes) == 0 {
		return nil, nil
	}
	var fieldRefs []Evaluator
	for _, f := range fields {
		fieldRefs = append(fieldRefs, NewDotExpr(f))
	}
	builder, err := builder.NewColumnBuilder(d.zctx, fields)
	if err != nil {
		return nil, err
	}
	cols := builder.TypedColumns(fieldTypes)
	typ, err := d.zctx.LookupTypeRecord(cols)
	if err != nil {
		return nil, err
	}
	return &dropper{typ, builder, fieldRefs}, nil
}

// complementFields returns the slice of fields and associated types
// that make up the complement of the set of fields in drops.
func complementFields(drops field.List, prefix field.Path, typ *zng.TypeRecord) (field.List, []zng.Type) {
	var fields field.List
	var types []zng.Type
	for _, c := range typ.Columns {
		if contains(drops, append(prefix, c.Name)) {
			continue
		}
		if typ, ok := c.Type.(*zng.TypeRecord); ok {
			innerFields, innerTypes := complementFields(drops, append(prefix, c.Name), typ)
			fields = append(fields, innerFields...)
			types = append(types, innerTypes...)
			continue
		}
		fields = append(fields, append(prefix, c.Name))
		types = append(types, c.Type)
	}
	return fields, types
}

func contains(ss field.List, el field.Path) bool {
	for _, s := range ss {
		if s.Equal(el) {
			return true
		}
	}
	return false
}

func (_ *Dropper) String() string { return "drop" }

func (_ *Dropper) Warning() string { return "" }

// Apply implements proc.Function and returns a new record comprising fields
// that are not specified in the set of drop targets.
func (d *Dropper) Apply(in *zng.Record) (*zng.Record, error) {
	id := in.Type.ID()
	dropper, ok := d.droppers[id]
	if !ok {
		var err error
		dropper, err = d.newDropper(in)
		if err != nil {
			return nil, err
		}
		d.droppers[id] = dropper
	}
	if dropper == nil {
		return nil, nil
	}
	return dropper.drop(in)
}

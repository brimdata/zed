package expr

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
	"golang.org/x/exp/slices"
)

type dropper struct {
	typ       zed.Type
	builder   *zed.RecordBuilder
	fieldRefs []Evaluator
}

func (d *dropper) drop(ectx Context, in *zed.Value) *zed.Value {
	if d.typ == in.Type {
		return in
	}
	b := d.builder
	b.Reset()
	for _, e := range d.fieldRefs {
		val := e.Eval(ectx, in)
		b.Append(val.Bytes)
	}
	zv, err := b.Encode()
	if err != nil {
		panic(err)
	}
	return zed.NewValue(d.typ, zv)
}

type Dropper struct {
	zctx      *zed.Context
	fields    field.List
	resolvers []Evaluator
	droppers  map[int]*dropper
}

func NewDropper(zctx *zed.Context, fields field.List) *Dropper {
	return &Dropper{
		zctx:     zctx,
		fields:   fields,
		droppers: make(map[int]*dropper),
	}
}

func (d *Dropper) newDropper(zctx *zed.Context, r *zed.Value) *dropper {
	fields, fieldTypes, match := complementFields(d.fields, nil, zed.TypeRecordOf(r.Type))
	if !match {
		// r.Type contains no fields matching d.fields, so we set
		// dropper.typ to r.Type to indicate that records of this type
		// should not be modified.
		return &dropper{typ: r.Type}
	}
	// If the set of dropped fields is equal to the all of record's
	// fields, then there is no output for this input type.
	// We return nil to block this input type.
	if len(fieldTypes) == 0 {
		return nil
	}
	var fieldRefs []Evaluator
	for _, f := range fields {
		fieldRefs = append(fieldRefs, NewDottedExpr(zctx, f))
	}
	builder, err := zed.NewRecordBuilder(d.zctx, fields)
	if err != nil {
		panic(err)
	}
	cols := builder.TypedColumns(fieldTypes)
	typ := d.zctx.MustLookupTypeRecord(cols)
	return &dropper{typ, builder, fieldRefs}
}

// complementFields returns the slice of fields and associated types that make
// up the complement of the set of fields in drops along with a boolean that is
// true if typ contains any the fields in drops.
func complementFields(drops field.List, prefix field.Path, typ *zed.TypeRecord) (field.List, []zed.Type, bool) {
	var fields field.List
	var types []zed.Type
	var match bool
	for _, c := range typ.Columns {
		fld := append(prefix, c.Name)
		if drops.Has(fld) {
			match = true
			continue
		}
		if typ, ok := zed.TypeUnder(c.Type).(*zed.TypeRecord); ok {
			if fs, ts, m := complementFields(drops, fld, typ); m {
				fields = append(fields, fs...)
				types = append(types, ts...)
				match = true
				continue
			}
		}
		fields = append(fields, slices.Clone(fld))
		types = append(types, c.Type)
	}
	return fields, types, match
}

func (_ *Dropper) String() string { return "drop" }

func (_ *Dropper) Warning() string { return "" }

func (d *Dropper) Eval(ectx Context, in *zed.Value) *zed.Value {
	if !zed.IsRecordType(in.Type) {
		return in
	}
	id := in.Type.ID()
	dropper, ok := d.droppers[id]
	if !ok {
		dropper = d.newDropper(d.zctx, in)
		d.droppers[id] = dropper
	}
	if dropper == nil {
		return d.zctx.Quiet()
	}
	return dropper.drop(ectx, in)
}

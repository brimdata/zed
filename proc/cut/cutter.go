package cut

import (
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

// A cutBuilder keeps the data structures needed for cutting one
// particular type of input record.
type cutBuilder struct {
	resolvers []expr.Evaluator
	builder   *proc.ColumnBuilder
	outType   *zng.TypeRecord
}

// cut returns a new record value from input record using the provided
// cutBuilder, or nil if the record can't be cut.
func (cb *cutBuilder) cut(in *zng.Record) *zng.Record {
	cb.builder.Reset()
	for _, resolver := range cb.resolvers {
		val, _ := resolver.Eval(in)
		cb.builder.Append(val.Bytes, val.IsContainer())
	}
	zv, err := cb.builder.Encode()
	if err != nil {
		// XXX internal error, what to do...
		return nil
	}
	return zng.NewRecord(cb.outType, zv)
}

type Cutter struct {
	zctx        *resolver.Context
	complement  bool
	cutBuilders map[int]*cutBuilder
	fields      []field.Static
	resolvers   []expr.Evaluator
	strict      bool
}

// NewCutter returns a Cutter for fieldnames. If complement is true,
// the Cutter copies fields that are not in fieldnames. If complement
// is false, the Cutter copies any fields in fieldnames, where targets
// specifies the copied field names.
func NewCutter(zctx *resolver.Context, complement bool, lhs []field.Static, rhs []expr.Evaluator) *Cutter {
	return &Cutter{
		zctx:        zctx,
		complement:  complement,
		fields:      lhs,
		resolvers:   rhs,
		cutBuilders: make(map[int]*cutBuilder),
	}
}

// NewStrictCutter is like NewCutter but, if complement is false, (*Cutter).Cut
// returns a record only if its input record contains all of the fields in lhs.
func NewStrictCutter(zctx *resolver.Context, complement bool, lhs []field.Static, rhs []expr.Evaluator) *Cutter {
	c := NewCutter(zctx, complement, lhs, rhs)
	c.strict = true
	return c
}

func (c *Cutter) FoundCut() bool {
	for _, ci := range c.cutBuilders {
		if ci != nil {
			return true
		}
	}
	return false
}

// complementBuilder creates a builder for the complement form of cut, where a
// all fields not in a set are to be cut from a record and passed on.
func (c *Cutter) complementBuilder(r *zng.Record) (*cutBuilder, error) {
	fields, fieldTypes := complementFields(c.fields, nil, r.Type)
	// if the set of cut -c fields is equal to the set of record
	// fields, then there is no output for this input type.
	if len(fieldTypes) == 0 {
		return nil, nil
	}

	var resolvers []expr.Evaluator
	for _, f := range fields {
		resolvers = append(resolvers, expr.NewDotExpr(f))
	}

	builder, err := proc.NewColumnBuilder(c.zctx, fields)
	if err != nil {
		return nil, err
	}
	cols := builder.TypedColumns(fieldTypes)
	outType, err := c.zctx.LookupTypeRecord(cols)
	if err != nil {
		return nil, err
	}
	return &cutBuilder{resolvers, builder, outType}, nil
}

// complementFields returns the slice of fields and associated types
// that make up the complement of the set of fields in drops.
func complementFields(drops []field.Static, prefix field.Static, typ *zng.TypeRecord) ([]field.Static, []zng.Type) {
	var fields []field.Static
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

func contains(ss []field.Static, el field.Static) bool {
	for _, s := range ss {
		if s.Equal(el) {
			return true
		}
	}
	return false
}

// setBuilder creates a builder for the regular form of cut, where a
// set of fields are to be cut from a record and passed on.
//
// Note that unlike for the complement form, we don't strictly need a
// different columnbuilder or set of resolvers per input type
// here. (We do need a different outType). Since the number of
// different input types is small wrt the number of input records, the
// optimization consisting of having a single columnbuilder and
// resolver set doesn't seem worth the added special casing.
func (c *Cutter) setBuilder(r *zng.Record) (*cutBuilder, error) {
	// Build up the output type.
	var fields []field.Static
	var resolvers []expr.Evaluator
	var outColTypes []zng.Type
	for i, resolver := range c.resolvers {
		val, err := resolver.Eval(r)
		if err != nil || val.Type == nil {
			// The field is absent, so for this input type, ...
			if c.strict {
				// ...produce no output.
				return nil, nil
			}
			// ...omit the field from the output.
			continue
		}
		fields = append(fields, c.fields[i])
		resolvers = append(resolvers, resolver)
		outColTypes = append(outColTypes, val.Type)
	}
	if len(fields) == 0 {
		return nil, nil
	}
	builder, err := proc.NewColumnBuilder(c.zctx, fields)
	if err != nil {
		return nil, err
	}
	cols := builder.TypedColumns(outColTypes)
	outType, err := c.zctx.LookupTypeRecord(cols)
	if err != nil {
		return nil, err
	}
	return &cutBuilder{resolvers, builder, outType}, nil
}

func (c *Cutter) builder(r *zng.Record) (*cutBuilder, error) {
	if c.complement {
		return c.complementBuilder(r)
	}
	return c.setBuilder(r)
}

// Cut returns a new record comprising fields copied from in according to the
// receiver's configuration.  If the resulting record would be empty, Cut
// returns nil.
func (c *Cutter) Cut(in *zng.Record) (*zng.Record, error) {
	cb, ok := c.cutBuilders[in.Type.ID()]
	if !ok {
		var err error
		cb, err = c.builder(in)
		if err != nil {
			return nil, err
		}
		c.cutBuilders[in.Type.ID()] = cb
	}
	if cb == nil {
		return nil, nil
	}
	return cb.cut(in), nil
}

package proc

import (
	"strings"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

// A cutBuilder keeps the data structures needed for cutting one
// particular type of input record.
type cutBuilder struct {
	resolvers []expr.FieldExprResolver
	builder   *ColumnBuilder
	outType   *zng.TypeRecord
}

// cut returns a new record value from input record using the provided
// cutBuilder, or nil if the record can't be cut.
func (cb *cutBuilder) cut(in *zng.Record) *zng.Record {
	cb.builder.Reset()
	for _, resolver := range cb.resolvers {
		val := resolver(in)
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
	fieldnames  []string
	strict      bool
}

// NewCutter returns a Cutter for fieldnames.  If complement is true, the Cutter
// copies fields that are not in fieldnames.  If complement is false, the Cutter
// copies fields that are in fieldnames.
func NewCutter(zctx *resolver.Context, complement bool, fieldnames []string) *Cutter {
	return &Cutter{
		zctx:        zctx,
		complement:  complement,
		fieldnames:  fieldnames,
		cutBuilders: make(map[int]*cutBuilder),
	}
}

// NewStrictCutter is like NewCutter but, if complement is false, (*Cutter).Cut
// returns a record only if its input record contains all of the fields in
// fieldnames.
func NewStrictCutter(zctx *resolver.Context, complement bool, fieldnames []string) *Cutter {
	c := NewCutter(zctx, complement, fieldnames)
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
	var resolvers []expr.FieldExprResolver
	var outColTypes []zng.Type

	iter := r.FieldIter()
	var fields []string
	for !iter.Done() {
		name, _, err := iter.Next()
		if err != nil {
			return nil, err
		}
		if !fieldIn(c.fieldnames, name) {
			fields = append(fields, name)
			resolver := expr.CompileFieldAccess(name)
			resolvers = append(resolvers, resolver)
			val := resolver(r)
			outColTypes = append(outColTypes, val.Type)
		}
	}
	// if the set of cut -c fields is equal to the set of record
	// fields, then there is no output for this input type.
	if len(outColTypes) == 0 {
		return nil, nil
	}
	builder, err := NewColumnBuilder(c.zctx, fields)
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

func fieldIn(set []string, cand string) bool {
	splits := strings.Split(cand, ".")
	for _, setel := range set {
		for j := range splits {
			prefix := strings.Join(splits[:j+1], ".")
			if prefix == setel {
				return true
			}
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
	var fields []string
	var resolvers []expr.FieldExprResolver
	var outColTypes []zng.Type
	for _, f := range c.fieldnames {
		resolver := expr.CompileFieldAccess(f)
		val := resolver(r)
		if val.Type == nil {
			// The field is absent, so for this input type, ...
			if c.strict {
				// ...produce no output.
				return nil, nil
			}
			// ...omit the field from the output.
			continue
		}
		fields = append(fields, f)
		resolvers = append(resolvers, resolver)
		outColTypes = append(outColTypes, val.Type)
	}
	if len(fields) == 0 {
		return nil, nil
	}
	builder, err := NewColumnBuilder(c.zctx, fields)
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

package proc

import (
	"fmt"
	"strings"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

// A cutBuilder keeps the data structures needed for cutting one
// particular type of input record.
type cutBuilder struct {
	resolvers []expr.FieldExprResolver
	builder   *ColumnBuilder
	outType   *zng.TypeRecord
}

type Cut struct {
	Base
	complement  bool
	cutBuilders map[int]*cutBuilder
	fieldnames  []string
}

// XXX update me
// Build the structures we need to construct output records efficiently.
// See the comment above for a description of the desired output.
// Note that we require any nested fields from the same parent record
// to be adjacent.  Alternatively we could re-order provided fields
// so the output record can be constructed efficiently, though we don't
// do this now since it might confuse users who expect to see output
// fields in the order they specified.
func CompileCutProc(c *Context, parent Proc, node *ast.CutProc) (*Cut, error) {
	// build this once at compile time for error checking.
	if !node.Complement {
		_, err := NewColumnBuilder(c.TypeContext, node.Fields)
		if err != nil {
			return nil, fmt.Errorf("compiling cut: %w", err)
		}
	}

	return &Cut{
		Base:        Base{Context: c, Parent: parent},
		complement:  node.Complement,
		cutBuilders: make(map[int]*cutBuilder),
		fieldnames:  node.Fields,
	}, nil
}

// cut returns a new record value from input record using the provided
// cutBuilder, or nil if the record can't be cut.
func (c *Cut) cut(cb *cutBuilder, in *zng.Record) *zng.Record {
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

	r, err := zng.NewRecord(cb.outType, zv)
	if err != nil {
		// records with invalid ts shouldn't get here
		return nil
	}
	return r
}

func (c *Cut) maybeWarn() {
	if c.complement {
		return
	}
	for _, ci := range c.cutBuilders {
		if ci != nil {
			return
		}
	}
	var msg string
	if len(c.fieldnames) == 1 {
		msg = fmt.Sprintf("Cut field %s not present in input", c.fieldnames[0])
	} else {
		msg = fmt.Sprintf("Cut fields %s not present together in input", strings.Join(c.fieldnames, ","))
	}
	c.Warnings <- msg
}

// complementBuilder creates a builder for the complement form of cut, where a
// all fields not in a set are to be cut from a record and passed on.
func (c *Cut) complementBuilder(r *zng.Record) (*cutBuilder, error) {
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
	builder, err := NewColumnBuilder(c.TypeContext, fields)
	if err != nil {
		return nil, err
	}
	cols := builder.TypedColumns(outColTypes)
	outType, err := c.TypeContext.LookupTypeRecord(cols)
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
func (c *Cut) setBuilder(r *zng.Record) (*cutBuilder, error) {
	var resolvers []expr.FieldExprResolver
	var outColTypes []zng.Type

	builder, err := NewColumnBuilder(c.TypeContext, c.fieldnames)
	if err != nil {
		return nil, err
	}
	for _, name := range c.fieldnames {
		resolvers = append(resolvers, expr.CompileFieldAccess(name))
	}

	// Build up the output type. If any of the cut fields
	// is absent, there is no output for this input type.
	for _, resolver := range resolvers {
		val := resolver(r)
		if val.Type == nil {
			return nil, nil
		}
		outColTypes = append(outColTypes, val.Type)
	}
	cols := builder.TypedColumns(outColTypes)
	outType, err := c.TypeContext.LookupTypeRecord(cols)
	if err != nil {
		return nil, err
	}
	return &cutBuilder{resolvers, builder, outType}, nil
}

func (c *Cut) builder(r *zng.Record) (*cutBuilder, error) {
	if c.complement {
		return c.complementBuilder(r)
	}
	return c.setBuilder(r)
}

func (c *Cut) Pull() (zbuf.Batch, error) {
	for {
		batch, err := c.Get()
		if EOS(batch, err) {
			c.maybeWarn()
			return nil, err
		}
		// Make new records with only the fields specified.
		// If a field specified doesn't exist, we don't include that record.
		// If the types change for the fields specified, we drop those records.
		recs := make([]*zng.Record, 0, batch.Length())
		for k := 0; k < batch.Length(); k++ {
			in := batch.Index(k)

			var cb *cutBuilder
			var ok bool
			if cb, ok = c.cutBuilders[in.Type.ID()]; !ok {
				cb, err = c.builder(in)
				if err != nil {
					return nil, err
				}
				c.cutBuilders[in.Type.ID()] = cb
			}

			if cb == nil {
				// One or more cut fields isn't present in this type of
				// input record, or the resulting record is empty (cut -c).
				// Either way, we drop this input.
				continue
			}

			out := c.cut(cb, in)
			if out != nil {
				recs = append(recs, out)
			}
		}
		span := batch.Span()
		batch.Unref()
		if len(recs) > 0 {
			return zbuf.NewArray(recs, span), nil
		}
	}
}

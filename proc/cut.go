package proc

import (
	"fmt"
	"strings"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

type Cut struct {
	Base
	resolvers []expr.FieldExprResolver
	builder   *ColumnBuilder
	cutmap    map[int]*zng.TypeRecord
	nblocked  int
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
	resolvers, err := expr.CompileFieldExprs(node.Fields)
	if err != nil {
		return nil, fmt.Errorf("compiling cut: %w", err)
	}
	var fields []string
	for _, field := range node.Fields {
		fields = append(fields, expr.FieldExprToString(field))
	}
	builder, err := NewColumnBuilder(c.TypeContext, fields)
	if err != nil {
		return nil, fmt.Errorf("compiling cut: %w", err)
	}
	return &Cut{
		Base:      Base{Context: c, Parent: parent},
		resolvers: resolvers,
		builder:   builder,
		cutmap:    make(map[int]*zng.TypeRecord),
	}, nil
}

// cut returns a new record value derived by keeping only the fields
// specified by name in the fields slice.  If the record can't be cut
// (i.e., it doesn't have one of the specified fields), returns nil.
func (c *Cut) cut(in *zng.Record) *zng.Record {
	// Check if we already have an output descriptor for this
	// input type
	typ, ok := c.cutmap[in.Type.ID()]
	if ok && typ == nil {
		// One or more cut fields isn't present in this type of
		// input record, drop it now.
		return nil
	}

	c.builder.Reset()
	var types []zng.Type
	if typ == nil {
		types = make([]zng.Type, 0, len(c.resolvers))
	}
	// Build the output record.  If we've already seen this input
	// record type, we don't care about the types, but if we haven't
	// gather the types as well so we can construct the output
	// descriptor.
	for _, resolver := range c.resolvers {
		val := resolver(in)
		if typ == nil {
			if val.Type == nil {
				// a field is missing... block this descriptor
				c.cutmap[in.Type.ID()] = nil
				c.nblocked++
				return nil
			}
			types = append(types, val.Type)
		}
		c.builder.Append(val.Bytes, val.IsContainer())
	}
	if typ == nil {
		cols := c.builder.TypedColumns(types)
		typ = c.TypeContext.LookupTypeRecord(cols)
		c.cutmap[in.Type.ID()] = typ
	}

	zv, err := c.builder.Encode()
	if err != nil {
		// XXX internal error, what to do...
		return nil
	}

	r, err := zng.NewRecord(typ, zv)
	if err != nil {
		// records with invalid ts shouldn't get here
		return nil
	}
	return r
}

func (c *Cut) warn() {
	if len(c.cutmap) > c.nblocked {
		return
	}
	names := c.builder.FullNames()
	var msg string
	if len(names) == 1 {
		msg = fmt.Sprintf("Cut field %s not present in input", names[0])
	} else {
		msg = fmt.Sprintf("Cut fields %s not present together in input", strings.Join(names, ","))
	}
	c.Warnings <- msg
}

func (c *Cut) Pull() (zbuf.Batch, error) {
	for {
		batch, err := c.Get()
		if EOS(batch, err) {
			c.warn()
			return nil, err
		}
		// Make new records with only the fields specified.
		// If a field specified doesn't exist, we don't include that record.
		// If the types change for the fields specified, we drop those records.
		recs := make([]*zng.Record, 0, batch.Length())
		for k := 0; k < batch.Length(); k++ {
			in := batch.Index(k)
			out := c.cut(in)
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

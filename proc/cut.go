package proc

import (
	"fmt"
	"strings"

	"github.com/mccanne/zq/ast"
	"github.com/mccanne/zq/expr"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
)

type Cut struct {
	Base
	resolvers []expr.FieldExprResolver
	builder   *ColumnBuilder
	cutmap    map[int]*zson.Descriptor
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
	resolvers, err := expr.CompileFieldExprArray(node.Fields)
	if err != nil {
		return nil, err
	}
	builder, err := NewColumnBuilder(node.Fields)
	if err != nil {
		return nil, err
	}
	return &Cut{
		Base:      Base{Context: c, Parent: parent},
		resolvers: resolvers,
		builder:   builder,
		cutmap:    make(map[int]*zson.Descriptor),
	}, nil
}

// cut returns a new record value derived by keeping only the fields
// specified by name in the fields slice.  If the record can't be cut
// (i.e., it doesn't have one of the specified fields), returns nil.
func (c *Cut) cut(in *zson.Record) *zson.Record {
	// Check if we already have an output descriptor for this
	// input type
	d, ok := c.cutmap[in.ID]
	if ok && d == nil {
		// One or more cut fields isn't present in this type of
		// input record, drop it now.
		return nil
	}

	c.builder.Reset()
	var types []zeek.Type
	if d == nil {
		types = make([]zeek.Type, 0, len(c.resolvers))
	}
	// Build the output record.  If we've already seen this input
	// record type, we don't care about the types, but if we haven't
	// gather the types as well so we can construct the output
	// descriptor.
	for _, resolver := range c.resolvers {
		val := resolver(in)
		if d == nil {
			if val.Type == nil {
				// a field is missing... block this descriptor
				c.cutmap[in.ID] = nil
				c.nblocked++
				return nil
			}
			types = append(types, val.Type)
		}
		c.builder.Append(val.Body)
	}
	if d == nil {
		cols := c.builder.TypedColumns(types)
		d = c.Resolver.GetByColumns(cols)
		c.cutmap[in.ID] = d
	}

	zv, err := c.builder.Encode()
	if err != nil {
		// XXX internal error, what to do...
		return nil
	}
	return zson.NewRecordNoTs(d, zv)
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

func (c *Cut) Pull() (zson.Batch, error) {
	batch, err := c.Get()
	if EOS(batch, err) {
		c.warn()
		return nil, err
	}
	defer batch.Unref()
	//
	// Make new records with only the fields specified.
	// If a field specified doesn't exist, we don't include that record.
	// if the types change for the fields specified, we drop those records.
	//
	recs := make([]*zson.Record, 0, batch.Length())
	for k := 0; k < batch.Length(); k++ {
		in := batch.Index(k)
		out := c.cut(in)
		if out != nil {
			recs = append(recs, out)
		}
	}
	if len(recs) == 0 {
		c.warn()
		return nil, nil
	}
	return zson.NewArray(recs, batch.Span()), nil
}

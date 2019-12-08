package proc

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zq"
	"github.com/mccanne/zq/pkg/zval"
)

var ErrNoField = errors.New("cut field not found")

type Cut struct {
	Base
	fields   []string
	cutmap   map[int]*zq.Descriptor
	nblocked int
}

func NewCut(c *Context, parent Proc, fields []string) *Cut {
	return &Cut{
		Base:   Base{Context: c, Parent: parent},
		fields: fields,
		cutmap: make(map[int]*zq.Descriptor),
	}
}

func (c *Cut) lookup(in *zq.Descriptor) *zq.Descriptor {
	d, ok := c.cutmap[in.ID]
	if ok {
		return d
	}
	var columns []zeek.Column
	for _, field := range c.fields {
		colno, ok := in.ColumnOfField(field)
		if !ok {
			// a field is missing... block this descriptor
			c.cutmap[in.ID] = nil
			c.nblocked++
			return nil
		}
		columns = append(columns, in.Type.Columns[colno])
	}
	out := c.Resolver.GetByColumns(columns)
	c.cutmap[in.ID] = out
	return out
}

// CreateCut returns a new record value derived by keeping only the fields
// specified by name in the fields slice.
func (c *Cut) cut(d *zq.Descriptor, in *zq.Record) (*zq.Record, error) {
	var zv zval.Encoding
	for _, column := range d.Type.Columns {
		// colno must exist for each field since the descriptor map
		// entry is only created when all the fields exist.
		colno, _ := in.ColumnOfField(column.Name)
		zv = zval.Append(zv, in.Slice(colno), zeek.IsContainerType(column.Type))
	}
	return zq.NewRecordNoTs(d, zv), nil
}

func (c *Cut) warn() {
	if len(c.cutmap) > c.nblocked {
		return
	}
	flds := strings.Join(c.fields, ",")
	plural := ""
	msg := "not present in input"
	if len(c.fields) > 1 {
		plural = "s"
		msg = "not present together in input"
	}
	c.Warnings <- fmt.Sprintf("Cut field%s %s %s", plural, flds, msg)
}

func (c *Cut) Pull() (zq.Batch, error) {
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
	recs := make([]*zq.Record, 0, batch.Length())
	for k := 0; k < batch.Length(); k++ {
		in := batch.Index(k)
		d := c.lookup(in.Descriptor)
		if d == nil {
			continue
		}
		out, err := c.cut(d, in)
		if err != nil {
			return nil, err
		}
		recs = append(recs, out)
	}
	if len(recs) == 0 {
		c.warn()
		return nil, nil
	}
	return zq.NewArray(recs, batch.Span()), nil
}

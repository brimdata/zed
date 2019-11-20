package proc

import (
	"fmt"

	"github.com/mccanne/zq/pkg/zson"
)

type Cut struct {
	Base
	fields     []string
	sawRecord  bool
	seenFields uint64
}

func NewCut(c *Context, parent Proc, fields []string) *Cut {
	return &Cut{Base: Base{Context: c, Parent: parent}, fields: fields}
}

// Check the bitmap of fields that we've seen.  If a requested field
// never appeared in the input, emit a warning about it.  This should be
// called once when this proc reaches the end of the stream.
func (c *Cut) WarnUnseen() {
	if !c.sawRecord {
		return
	}
	for i, name := range c.fields {
		if (c.seenFields & (1 << i)) == 0 {
			c.Warnings <- fmt.Sprintf("Cut field %s not present in input", name)
		}
	}
}

func (c *Cut) Pull() (zson.Batch, error) {
	batch, err := c.Get()
	if EOS(batch, err) {
		c.WarnUnseen()
		return nil, err
	}
	defer batch.Unref()
	c.sawRecord = true
	//
	// Make new records with only the fields specified.
	// If a field specified doesn't exist, we don't include that record.
	// if the types change for the fields specified, we drop those records.
	//
	out := make([]*zson.Record, 0, batch.Length())
	var d *zson.Descriptor
	for k := 0; k < batch.Length(); k++ {
		in := batch.Index(k)
		r, found, err := c.Resolver.CreateCut(in, c.fields)
		c.seenFields |= found
		if err != nil {
			c.WarnUnseen()
			return nil, err
		}
		if r != nil {
			if d == nil {
				d = r.Descriptor
			}
			// Check that the types are the same throughout
			// if any of the types change, we ignore those records.
			if d == r.Descriptor {
				out = append(out, r)
			}
		}
	}
	if d == nil {
		c.WarnUnseen()
		return nil, nil
	}
	//XXX we should compute a new span here because some records may be dropped
	return zson.NewArray(out, batch.Span()), nil
}

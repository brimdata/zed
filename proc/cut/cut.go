package cut

import (
	"fmt"
	"strings"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

type Proc struct {
	proc.Parent
	complement bool
	fieldnames []string
	cutter     *Cutter
}

// XXX update me
// Build the structures we need to construct output records efficiently.
// See the comment above for a description of the desired output.
// Note that we require any nested fields from the same parent record
// to be adjacent.  Alternatively we could re-order provided fields
// so the output record can be constructed efficiently, though we don't
// do this now since it might confuse users who expect to see output
// fields in the order they specified.
func New(parent proc.Parent, node *ast.CutProc) (*Proc, error) {
	var fieldnames, targets []string
	for _, fa := range node.Fields {
		if fa.Target == "" {
			fa.Target = fa.Source
		}
		targets = append(targets, fa.Target)
		fieldnames = append(fieldnames, fa.Source)
	}
	// build this once at compile time for error checking.
	if !node.Complement {
		_, err := proc.NewColumnBuilder(parent.TypeContext, targets)
		if err != nil {
			return nil, fmt.Errorf("compiling cut: %w", err)
		}
	}

	return &Proc{
		Parent:     parent,
		complement: node.Complement,
		fieldnames: fieldnames,
		cutter:     NewCutter(parent.TypeContext, node.Complement, targets, fieldnames),
	}, nil
}

func (c *Proc) maybeWarn() {
	if c.complement || c.cutter.FoundCut() {
		return
	}
	var msg string
	if len(c.fieldnames) == 1 {
		msg = fmt.Sprintf("Cut field %s not present in input", c.fieldnames[0])
	} else {
		msg = fmt.Sprintf("Cut fields %s not present together in input", strings.Join(c.fieldnames, ","))
	}
	c.Warnings <- msg
}

func (c *Proc) Pull() (zbuf.Batch, error) {
	for {
		batch, err := c.Get()
		if proc.EOS(batch, err) {
			c.maybeWarn()
			return nil, err
		}
		// Make new records with only the fields specified.
		// If a field specified doesn't exist, we don't include that record.
		// If the types change for the fields specified, we drop those records.
		recs := make([]*zng.Record, 0, batch.Length())
		for k := 0; k < batch.Length(); k++ {
			in := batch.Index(k)

			out, err := c.cutter.Cut(in)
			if err != nil {
				return nil, err
			}

			if out != nil {
				recs = append(recs, out)
			}
		}
		batch.Unref()
		if len(recs) > 0 {
			return zbuf.NewArray(recs), nil
		}
	}
}

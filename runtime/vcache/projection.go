package vcache

import (
	"fmt"
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio"
	"golang.org/x/sync/errgroup"
)

type Projection struct {
	object  *Object
	cuts    []*cut
	off     int
	builder zcode.Builder
	val     zed.Value
}

type cut struct {
	it  iterator
	typ zed.Type
}

var _ zio.Reader = (*Projection)(nil)

func NewProjection(o *Object, names []string) (*Projection, error) {
	cuts, err := findCuts(o, names)
	if err != nil {
		return nil, err
	}
	return &Projection{
		object: o,
		cuts:   cuts,
	}, nil
}

func (p *Projection) Read() (*zed.Value, error) {
	o := p.object
	var c *cut
	for c == nil {
		if p.off >= len(o.typeIDs) {
			return nil, nil
		}
		id := o.typeIDs[p.off]
		p.off++
		c = p.cuts[id]
	}
	p.builder.Truncate()
	if err := c.it(&p.builder); err != nil {
		return nil, err
	}
	p.val = *zed.NewValue(c.typ, p.builder.Bytes().Body())
	return &p.val, nil
}

func findCuts(o *Object, names []string) ([]*cut, error) {
	var dirty bool
	cuts := make([]*cut, len(o.types))
	var group errgroup.Group
	group.SetLimit(-1)
	// Loop through each type to determine if there is a cut and build
	// a cut for that type.  The creation of all the iterators is done
	// in parallel to avoid synchronous round trips to storage.
	for k, typ := range o.types {
		recType := zed.TypeRecordOf(typ)
		if recType == nil {
			continue
		}
		fields := Under(o.vectors[k]).(Record)
		var actuals []string
		for _, name := range names {
			if _, ok := recType.ColumnOfField(name); !ok {
				continue
			}
			actuals = append(actuals, name)
		}
		if len(actuals) == 0 {
			continue
		}
		dirty = true
		whichCut := k
		group.Go(func() error {
			c, err := newCut(o.local, recType, fields, actuals, o.reader)
			cuts[whichCut] = c
			return err
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	if !dirty {
		return nil, fmt.Errorf("none of the specified fields were found: %s", strings.Join(names, ", "))
	}
	return cuts, nil
}

func newCut(zctx *zed.Context, typ *zed.TypeRecord, fields []Vector, actuals []string, reader storage.Reader) (*cut, error) {
	var group errgroup.Group
	group.SetLimit(-1)
	iters := make([]iterator, len(actuals))
	columns := make([]zed.Column, len(actuals))
	for k, name := range actuals {
		col, _ := typ.ColumnOfField(name)
		columns[k] = typ.Columns[col]
		which := k
		group.Go(func() error {
			it, err := fields[col].NewIter(reader)
			if err != nil {
				return err
			}
			iters[which] = it
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	outType, err := zctx.LookupTypeRecord(columns)
	if err != nil {
		return nil, err
	}
	project := func(b *zcode.Builder) error {
		b.BeginContainer()
		for _, it := range iters {
			if err := it(b); err != nil {
				return err
			}
		}
		b.EndContainer()
		return nil
	}
	return &cut{it: project, typ: outType}, nil
}

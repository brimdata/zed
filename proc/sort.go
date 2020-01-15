package proc

import (
	"fmt"

	"github.com/mccanne/zq/expr"
	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zng"
)

type Sort struct {
	Base
	dir        int
	limit      int
	nullsFirst bool
	fields     []expr.FieldExprResolver
	out        []*zbuf.Record
}

// defaultSortLimit is the default limit of the number of records that
// sort will process and, otherwise, return an error if this limit is exceeded.
// The value can be overridden by setting the limit param on the SortProc.
const defaultSortLimit = 1000000

func NewSort(c *Context, parent Proc, limit int, fields []expr.FieldExprResolver, dir int, nullsFirst bool) *Sort {
	if limit == 0 {
		limit = defaultSortLimit
	}
	return &Sort{Base: Base{Context: c, Parent: parent}, dir: dir, limit: limit, nullsFirst: nullsFirst, fields: fields}
}

func firstOf(d *zbuf.Descriptor, which zng.Type) string {
	for _, col := range d.Type.Columns {
		if zng.SameType(col.Type, which) {
			return col.Name
		}
	}
	return ""
}

func firstNot(d *zbuf.Descriptor, which zng.Type) string {
	for _, col := range d.Type.Columns {
		if !zng.SameType(col.Type, which) {
			return col.Name
		}
	}
	return ""
}

func guessSortField(rec *zbuf.Record) string {
	d := rec.Descriptor
	if fld := firstOf(d, zng.TypeCount); fld != "" {
		return fld
	}
	if fld := firstOf(d, zng.TypeInt); fld != "" {
		return fld
	}
	if fld := firstOf(d, zng.TypeDouble); fld != "" {
		return fld
	}
	if fld := firstNot(d, zng.TypeTime); fld != "" {
		return fld
	}
	return "ts"
}

func (s *Sort) Pull() (zbuf.Batch, error) {
	for {
		batch, err := s.Get()
		if err != nil {
			return nil, err
		}
		if batch == nil {
			return s.sort(), nil
		}
		defer batch.Unref()
		if len(s.out)+batch.Length() > s.limit {
			return nil, fmt.Errorf("sort limit hit (%d)", s.limit)
		}
		// XXX this should handle group-by every ... need to change how we do this
		s.consume(batch)
	}
}

func (s *Sort) consume(batch zbuf.Batch) {
	//XXX this could be made more efficient
	for k := 0; k < batch.Length(); k++ {
		s.out = append(s.out, batch.Index(k).Keep())
	}
}

func (s *Sort) sort() zbuf.Batch {
	out := s.out
	if len(out) == 0 {
		return nil
	}
	s.out = nil
	if s.fields == nil {
		fld := guessSortField(out[0])
		resolver := func(r *zbuf.Record) zng.Value {
			e, err := r.Access(fld)
			if err != nil {
				return zng.Value{}
			}
			return e
		}
		s.fields = []expr.FieldExprResolver{resolver}
	}
	nullsMax := !s.nullsFirst
	if s.dir < 0 {
		nullsMax = !nullsMax
	}
	sorter := expr.NewSortFn(nullsMax, s.fields...)
	sortWithDir := func(a, b *zbuf.Record) int {
		return s.dir * sorter(a, b)
	}
	expr.SortStable(out, sortWithDir)
	return zbuf.NewArray(out, nano.NewSpanTs(s.MinTs, s.MaxTs))
}

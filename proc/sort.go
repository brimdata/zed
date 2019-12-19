package proc

import (
	"fmt"

	"github.com/mccanne/zq/expr"
	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zng"
)

type Sort struct {
	Base
	dir    int
	limit  int
	fields []expr.FieldExprResolver
	out    []*zng.Record
}

// defaultSortLimit is the default limit of the number of records that
// sort will process and, otherwise, return an error if this limit is exceeded.
// The value can be overridden by setting the limit param on the SortProc.
const defaultSortLimit = 1000000

func NewSort(c *Context, parent Proc, limit int, fields []expr.FieldExprResolver, dir int) *Sort {
	if limit == 0 {
		limit = defaultSortLimit
	}
	return &Sort{Base: Base{Context: c, Parent: parent}, dir: dir, limit: limit, fields: fields}
}

func firstOf(d *zng.Descriptor, which zeek.Type) string {
	for _, col := range d.Type.Columns {
		if zeek.SameType(col.Type, which) {
			return col.Name
		}
	}
	return ""
}

func firstNot(d *zng.Descriptor, which zeek.Type) string {
	for _, col := range d.Type.Columns {
		if !zeek.SameType(col.Type, which) {
			return col.Name
		}
	}
	return ""
}

func guessSortField(rec *zng.Record) string {
	d := rec.Descriptor
	if fld := firstOf(d, zeek.TypeCount); fld != "" {
		return fld
	}
	if fld := firstOf(d, zeek.TypeInt); fld != "" {
		return fld
	}
	if fld := firstOf(d, zeek.TypeDouble); fld != "" {
		return fld
	}
	if fld := firstNot(d, zeek.TypeTime); fld != "" {
		return fld
	}
	return "ts"
}

func (s *Sort) Pull() (zng.Batch, error) {
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

func (s *Sort) consume(batch zng.Batch) {
	//XXX this could be made more efficient
	for k := 0; k < batch.Length(); k++ {
		s.out = append(s.out, batch.Index(k).Keep())
	}
}

func (s *Sort) sort() zng.Batch {
	out := s.out
	if len(out) == 0 {
		return nil
	}
	s.out = nil
	if s.fields == nil {
		fld := guessSortField(out[0])
		resolver := func(r *zng.Record) zeek.TypedEncoding {
			e, err := r.Access(fld)
			if err != nil {
				return zeek.TypedEncoding{}
			}
			return e
		}
		s.fields = []expr.FieldExprResolver{resolver}
	}
	sorter := expr.NewSortFn(true, s.fields...)
	sortWithDir := func(a, b *zng.Record) int {
		return s.dir * sorter(a, b)
	}
	expr.SortStable(out, sortWithDir)
	return zng.NewArray(out, nano.NewSpanTs(s.MinTs, s.MaxTs))
}

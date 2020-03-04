package proc

import (
	"fmt"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

type Sort struct {
	Base
	dir        int
	limit      int
	nullsFirst bool
	fields     []expr.FieldExprResolver
	out        []*zng.Record
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

func firstOf(typ *zng.TypeRecord, which []zng.Type) string {
	for _, col := range typ.Columns {
		for _, t := range which {
			if zng.SameType(col.Type, t) {
				return col.Name
			}
		}
	}
	return ""
}

func firstNot(typ *zng.TypeRecord, which zng.Type) string {
	for _, col := range typ.Columns {
		if !zng.SameType(col.Type, which) {
			return col.Name
		}
	}
	return ""
}

var intTypes = []zng.Type{
	zng.TypeInt16,
	zng.TypeUint16,
	zng.TypeInt32,
	zng.TypeUint32,
	zng.TypeInt64,
	zng.TypeUint64,
}

func guessSortField(rec *zng.Record) string {
	typ := rec.Type
	if fld := firstOf(typ, intTypes); fld != "" {
		return fld
	}
	if fld := firstOf(typ, []zng.Type{zng.TypeFloat64}); fld != "" {
		return fld
	}
	if fld := firstNot(typ, zng.TypeTime); fld != "" {
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
		if len(s.out)+batch.Length() > s.limit {
			batch.Unref()
			return nil, fmt.Errorf("sort limit hit (%d)", s.limit)
		}
		// XXX this should handle group-by every ... need to change how we do this
		s.consume(batch)
		batch.Unref()
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
		resolver := func(r *zng.Record) zng.Value {
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
	sortWithDir := func(a, b *zng.Record) int {
		return s.dir * sorter(a, b)
	}
	expr.SortStable(out, sortWithDir)
	return zbuf.NewArray(out, nano.NewSpanTs(s.MinTs, s.MaxTs))
}

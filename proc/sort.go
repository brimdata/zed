package proc

import (
	"fmt"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

type ErrSortLimitReached int

func (e ErrSortLimitReached) Error() string {
	return fmt.Sprintf("sort limit hit (%d)", e)
}

type Sort struct {
	Base
	dir        int
	limit      int
	nullsFirst bool
	fields     []ast.FieldExpr
	resolvers  []expr.FieldExprResolver
	out        []*zng.Record
}

// defaultSortLimit is the default limit of the number of records that
// sort will process and, otherwise, return an error if this limit is exceeded.
// The value can be overridden by setting the limit param on the SortProc.
const defaultSortLimit = 1000000

func CompileSortProc(c *Context, parent Proc, node *ast.SortProc) (*Sort, error) {
	limit := node.Limit
	if limit == 0 {
		limit = defaultSortLimit
	}
	resolvers, err := expr.CompileFieldExprs(node.Fields)
	if err != nil {
		return nil, err
	}
	return &Sort{
		Base:       Base{Context: c, Parent: parent},
		dir:        node.SortDir,
		limit:      limit,
		nullsFirst: node.NullsFirst,
		fields:     node.Fields,
		resolvers:  resolvers,
	}, nil
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
			return nil, ErrSortLimitReached(s.limit)
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
	if s.resolvers == nil {
		fld := guessSortField(out[0])
		resolver := func(r *zng.Record) zng.Value {
			e, err := r.Access(fld)
			if err != nil {
				return zng.Value{}
			}
			return e
		}
		s.resolvers = []expr.FieldExprResolver{resolver}
	} else {
		s.warnAboutUnseenFields(out)
	}
	nullsMax := !s.nullsFirst
	if s.dir < 0 {
		nullsMax = !nullsMax
	}
	sorter := expr.NewSortFn(nullsMax, s.resolvers...)
	sortWithDir := func(a, b *zng.Record) int {
		return s.dir * sorter(a, b)
	}
	expr.SortStable(out, sortWithDir)
	return zbuf.NewArray(out, nano.NewSpanTs(s.MinTs, s.MaxTs))
}

func (s *Sort) warnAboutUnseenFields(records []*zng.Record) {
	unseenFields := make(map[ast.FieldExpr]expr.FieldExprResolver)
	for i, r := range s.resolvers {
		unseenFields[s.fields[i]] = r
	}
	sawType := make(map[*zng.TypeRecord]bool)
	for _, rec := range records {
		if !sawType[rec.Type] {
			sawType[rec.Type] = true
			for field, res := range unseenFields {
				if !res(rec).IsNil() {
					delete(unseenFields, field)
				}
			}
			if len(unseenFields) == 0 {
				break
			}
		}
	}
	for _, f := range s.fields {
		if _, ok := unseenFields[f]; ok {
			s.Warnings <- fmt.Sprintf("Sort field %s not present in input", expr.FieldExprToString(f))
		}
	}
}

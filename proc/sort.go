package proc

import (
	"fmt"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
)

// SortProc xxx
type SortProc struct {
	Base
	dir    int
	limit  int
	fields []string
	out    []*zson.Record
}

// defaultSortLimit is the default limit of the number of records that
// sort will process and, otherwise, return an error if this limit is exceeded.
// The value can be overridden by setting the limit param on the SortProc.
const defaultSortLimit = 1000000

func NewSortProc(c *Context, parent Proc, limit int, fields []string, dir int) *SortProc {
	if limit == 0 {
		limit = defaultSortLimit
	}
	return &SortProc{Base: Base{Context: c, Parent: parent}, dir: dir, limit: limit, fields: fields}
}

func firstOf(d *zson.Descriptor, which zeek.Type) string {
	for _, col := range d.Type.Columns {
		if zeek.SameType(col.Type, which) {
			return col.Name
		}
	}
	return ""
}

func firstNot(d *zson.Descriptor, which zeek.Type) string {
	for _, col := range d.Type.Columns {
		if !zeek.SameType(col.Type, which) {
			return col.Name
		}
	}
	return ""
}

func guessSortField(rec *zson.Record) string {
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

func (s *SortProc) Pull() (zson.Batch, error) {
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

func (s *SortProc) consume(batch zson.Batch) {
	//XXX this could be made more efficient
	for k := 0; k < batch.Length(); k++ {
		s.out = append(s.out, batch.Index(k).Keep())
	}
}

func (s *SortProc) sort() zson.Batch {
	out := s.out
	if len(out) == 0 {
		return nil
	}
	s.out = nil
	if s.fields == nil {
		s.fields = []string{guessSortField(out[0])}
	}
	sorter := zson.NewSortFn(s.dir, s.fields...)
	zson.SortStable(out, sorter)
	return zson.NewArray(out, nano.NewSpanTs(s.MinTs, s.MaxTs))
}

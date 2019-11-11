package proc

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
)

// SortProc xxx
type SortProc struct {
	Base
	dir     int
	limit   int
	fields  []string
	sorters []SortFn
	out     []*zson.Record
}

// defaultSortLimit is the default limit of the number of records that
// sort will process and, otherwise, return an error if this limit is exceeded.
// The value can be overridden by setting the limit param on the SortProc.
const defaultSortLimit = 1000000

func NewSortProc(c *Context, parent Proc, limit int, fields []string, dir int) *SortProc {
	var sorters []SortFn
	if fields == nil {
		sorters = make([]SortFn, 1)
	} else {
		sorters = make([]SortFn, len(fields))
	}
	if limit == 0 {
		limit = defaultSortLimit
	}
	return &SortProc{Base{Context: c, Parent: parent}, dir, limit, fields, sorters, nil}
}

func lookupSorter(r *zson.Record, field string, dir int) SortFn {
	zv := r.ValueByField(field)
	if zv == nil {
		return nil
	}
	typ := zv.Type()
	switch typ.(type) {
	default:
		return func(*zson.Record, *zson.Record, int) int { return 1 }
	case *zeek.TypeOfBool:
		return func(a, b *zson.Record, dir int) int {
			va, err := a.AccessBool(field)
			if err != nil {
				return -dir
			}
			vb, err := b.AccessBool(field)
			if err != nil {
				return dir
			}
			if va == vb {
				return 0
			}
			if va {
				return dir
			}
			return -dir
		}

	case *zeek.TypeOfString, *zeek.TypeOfEnum:
		return func(a, b *zson.Record, dir int) int {
			va, _, err := a.Access(field)
			if err != nil {
				return -dir
			}
			vb, _, err := b.Access(field)
			if err != nil {
				return dir
			}
			delta := bytes.Compare(va, vb)
			if delta < 0 {
				return -dir
			} else if delta > 0 {
				return dir
			}
			return 0
		}

		//XXX note zeek port type can have "/tcp" etc suffix according
		// to docs but we've only encountered ints in data files.
		// need to fix this.  XXX also we should break this sorts
		// into the different types.
	case *zeek.TypeOfPort, *zeek.TypeOfCount, *zeek.TypeOfInt:
		return func(a, b *zson.Record, dir int) int {
			va, err := a.AccessInt(field)
			if err != nil {
				return -dir
			}
			vb, err := b.AccessInt(field)
			if err != nil {
				return dir
			}
			if va < vb {
				return -dir
			} else if va > vb {
				return dir
			}
			return 0
		}

	case *zeek.TypeOfDouble:
		return func(a, b *zson.Record, dir int) int {
			va, err := a.AccessDouble(field)
			if err != nil {
				return -dir
			}
			vb, err := b.AccessDouble(field)
			if err != nil {
				return dir
			}
			if va < vb {
				return -dir
			} else if va > vb {
				return dir
			}
			return 0
		}

	case *zeek.TypeOfTime, *zeek.TypeOfInterval:
		return func(a, b *zson.Record, dir int) int {
			va, err := a.AccessTime(field)
			if err != nil {
				return -dir
			}
			vb, err := b.AccessTime(field)
			if err != nil {
				return dir
			}
			if va < vb {
				return -dir
			} else if va > vb {
				return dir
			}
			return 0
		}

	case *zeek.TypeOfAddr:
		return func(a, b *zson.Record, dir int) int {
			va, err := a.AccessIP(field)
			if err != nil {
				return -dir
			}
			vb, err := b.AccessIP(field)
			if err != nil {
				return dir
			}
			return bytes.Compare(va.To16(), vb.To16()) * dir
		}
	}
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

func guess(recs []*zson.Record) string {
	d := recs[0].Descriptor
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

type SortFn func(*zson.Record, *zson.Record, int) int

func (s *SortProc) sorter(recs []*zson.Record) func(int, int) bool {
	return func(i, j int) bool {
		if s.fields == nil {
			// If no sort-by fields are given, then we try to guess
			// something that makes sense...
			fld := guess(recs)
			s.fields = []string{fld}
		}
		for k, field := range s.fields {
			sf := s.sorters[k]
			if sf == nil {
				sf = lookupSorter(recs[i], field, s.dir)
				if sf == nil {
					// if we can't build a sorter, then
					// the record doesn't have a field
					// with the corresponding name, so
					// we return true causing these records
					// to get sorted first
					return true
				}
				s.sorters[k] = sf
			}
			v := s.sorters[k](recs[i], recs[j], s.dir)
			// If the events don't match, then return the sort
			// info.  Otherwise, they match and we continue on
			// on in the loop to the secondary key, etc.
			if v != 0 {
				return v < 0
			}
		}
		// All the keys matched with equality, so arbitrarily return true.
		return true
	}
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

//XXX this is just string sorting for now on the first field.
// also, need to sort according to types
func (s *SortProc) sortRecords(recs []*zson.Record) {
	sorter := s.sorter(recs)
	sort.SliceStable(recs, sorter)
}

func (s *SortProc) sort() zson.Batch {
	out := s.out
	if len(out) == 0 {
		return nil
	}
	s.out = nil
	s.sortRecords(out)
	return zson.NewArray(out, nano.NewSpanTs(s.MinTs, s.MaxTs))
}

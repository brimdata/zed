package zson

import (
	"bytes"
	"sort"

	"github.com/mccanne/zq/pkg/zeek"
)

type Sorter struct {
	records []*Record
	fields  []string
	sorters []sortFn
	dir     int
}

type sortFn func(a *Record, b *Record, dir int) int

func NewSorter(dir int, fields ...string) *Sorter {
	return &Sorter{
		dir:     dir,
		fields:  fields,
		sorters: make([]sortFn, len(fields)),
	}
}

// SortStable performs a stable sort on the provided records.
func (s *Sorter) SortStable(records []*Record) {
	s.records = records
	sort.Stable(s)
}

// Swap implements sort.Interface for *Record slices.
func (s *Sorter) Len() int { return len(s.records) }

// Swap implements sort.Interface for *Record slices.
func (s *Sorter) Swap(i, j int) { s.records[i], s.records[j] = s.records[j], s.records[i] }

// Less implements sort.Interface for *Record slices.
func (s *Sorter) Less(i, j int) bool {
	for k, field := range s.fields {
		sf := s.sorters[k]
		if sf == nil {
			sf = lookupSorter(s.records[i], field, s.dir)
		}
		if sf == nil {
			// if we can't build a sorter, then
			// the record doesn't have a field
			// with the corresponding name, so
			// we return true causing these records
			// to get sorted first
			return true
		}
		s.sorters[k] = sf
		v := sf(s.records[i], s.records[j], s.dir)
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

// Push adds x as element Len(). Implements heap.Interface.
func (s *Sorter) Push(r interface{}) {
	s.records = append(s.records, r.(*Record))
}

// Pop removes the first element in the array. Implements heap.Interface.
func (s *Sorter) Pop() interface{} {
	rec := s.records[len(s.records)-1]
	s.records = s.records[:len(s.records)-1]
	return rec
}

// Index returns the ith record.
func (s *Sorter) Index(i int) *Record {
	return s.records[i]
}

func lookupSorter(r *Record, field string, dir int) sortFn {
	zv := r.ValueByField(field)
	if zv == nil {
		return nil
	}
	typ := zv.Type()
	switch typ.(type) {
	default:
		return func(*Record, *Record, int) int { return 1 }
	case *zeek.TypeOfBool:
		return func(a, b *Record, dir int) int {
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
		return func(a, b *Record, dir int) int {
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
		return func(a, b *Record, dir int) int {
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
		return func(a, b *Record, dir int) int {
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
		return func(a, b *Record, dir int) int {
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
		return func(a, b *Record, dir int) int {
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

package expr

import (
	"bytes"
	"sort"

	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
)

type SortFn func(a *zson.Record, b *zson.Record) int

// Internal function that compares two values of compatible types.
type comparefn func(a, b []byte) int

func rawcompare(a, b []byte, dir int) int {
	v := bytes.Compare(a, b)
	if v < 0 {
		return -dir
	} else {
		return dir
	}
}

func NewSortFn(dir int, fields ...FieldExprResolver) SortFn {
	sorters := make(map[*zeek.Type]comparefn)
	return func(a *zson.Record, b *zson.Record) int {
		for _, resolver := range fields {
			typea, vala := resolver(a)
			typeb, valb := resolver(b)

			// Nil types indicate a field isn't present, sort
			// these records last
			if typea == nil && typeb == nil { return 0 }
			if typea == nil { return 1 }
			if typeb == nil { return -1 }

			// If values are of different types, just compare
			// the string representation of the type
			if !zeek.SameType(typea, typeb) {
				return rawcompare([]byte(typea.String()), []byte(typeb.String()), dir)
			}

			sf, ok := sorters[&typea]
			if !ok {
				sf = lookupSorter(typea, dir)
				sorters[&typea] = sf
			}

			v := sf(vala, valb)
			// If the events don't match, then return the sort
			// info.  Otherwise, they match and we continue on
			// on in the loop to the secondary key, etc.
			if v != 0 {
				return v
			}
		}
		// All the keys matched with equality.
		return 0
	}
}

// SortStable performs a stable sort on the provided records.
func SortStable(records []*zson.Record, sorter SortFn) {
	slice := &RecordSlice{records, sorter}
	sort.Stable(slice)
}

type RecordSlice struct {
	records []*zson.Record
	sorter  SortFn
}

func NewRecordSlice(sorter SortFn) *RecordSlice {
	return &RecordSlice{sorter: sorter}
}

// Swap implements sort.Interface for *Record slices.
func (s *RecordSlice) Len() int { return len(s.records) }

// Swap implements sort.Interface for *Record slices.
func (s *RecordSlice) Swap(i, j int) { s.records[i], s.records[j] = s.records[j], s.records[i] }

// Less implements sort.Interface for *Record slices.
func (s *RecordSlice) Less(i, j int) bool {
	return s.sorter(s.records[i], s.records[j]) <= 0
}

// Push adds x as element Len(). Implements heap.Interface.
func (s *RecordSlice) Push(r interface{}) {
	s.records = append(s.records, r.(*zson.Record))
}

// Pop removes the first element in the array. Implements heap.Interface.
func (s *RecordSlice) Pop() interface{} {
	rec := s.records[len(s.records)-1]
	s.records = s.records[:len(s.records)-1]
	return rec
}

// Index returns the ith record.
func (s *RecordSlice) Index(i int) *zson.Record {
	return s.records[i]
}

func lookupSorter(typ zeek.Type, dir int) comparefn {
	switch typ {
	default:
		return func(a, b []byte) int {
			return rawcompare(a, b, dir)
		}
	case zeek.TypeBool:
		return func(a, b []byte) int {
			va, err := zeek.TypeBool.Parse(a)
			if err != nil {
				return -dir
			}
			vb, err := zeek.TypeBool.Parse(b)
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

	case zeek.TypeString, zeek.TypeEnum:
		return func(a, b []byte) int {
			return rawcompare(a, b, dir)
		}

	//XXX note zeek port type can have "/tcp" etc suffix according
	// to docs but we've only encountered ints in data files.
	// need to fix this.  XXX also we should break this sorts
	// into the different types.
	case zeek.TypePort, zeek.TypeCount, zeek.TypeInt:
		return func(a, b []byte) int {
			va, err := zeek.TypeInt.Parse(a)
			if err != nil {
				return -dir
			}
			vb, err := zeek.TypeInt.Parse(b)
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

	case zeek.TypeDouble:
		return func(a, b []byte) int {
			va, err := zeek.TypeDouble.Parse(a)
			if err != nil {
				return -dir
			}
			vb, err := zeek.TypeDouble.Parse(b)
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

	case zeek.TypeTime, zeek.TypeInterval:
		return func(a, b []byte) int {
			va, err := zeek.TypeTime.Parse(a)
			if err != nil {
				return -dir
			}
			vb, err := zeek.TypeTime.Parse(b)
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

	case zeek.TypeAddr:
		return func(a, b []byte) int {
			va, err := zeek.TypeAddr.Parse(a)
			if err != nil {
				return -dir
			}
			vb, err := zeek.TypeAddr.Parse(b)
			if err != nil {
				return dir
			}
			return bytes.Compare(va.To16(), vb.To16()) * dir
		}
	}
}

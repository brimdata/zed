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

func isUnset(val zeek.TypedEncoding) bool {
	if val.Body == nil || zeek.SameType(val.Type, zeek.TypeUnset) {
		return true
	}
	return false
}

// NewSortFn creates a function that compares a pair of Records
// based on the provided ordered list of fields.
// The returned function uses the same return conventions as standard
// routines such as bytes.Compare() and strings.Compare(), so it may
// be used with packages such as heap and sort.
// A record in which one of the comparison fields is not present is
// considered to be smaller than a record in which the field is present.
// The handling of records in which a comparison field is unset is
// governed by the unsetMax parameter.  If this parameter is true,
// a record with unset is considered larger than a record with any other
// value, and vice versa.
func NewSortFn(unsetMax bool, fields ...FieldExprResolver) SortFn {
	sorters := make(map[zeek.Type]comparefn)
	return func(ra *zson.Record, rb *zson.Record) int {
		for _, resolver := range fields {
			a := resolver(ra)
			b := resolver(rb)

			// Nil types indicate a field isn't present, sort
			// these records to the minimum value so they appear
			// first in sort output.
			if a.Type == nil && b.Type == nil {
				return 0
			}
			if a.Type == nil {
				return -1
			}
			if b.Type == nil {
				return 1
			}

			// Handle unset according to unsetMax
			unsetA := isUnset(a)
			unsetB := isUnset(b)
			if unsetA && unsetB {
				return 0
			}
			if unsetA {
				if unsetMax {
					return 1
				} else {
					return -1
				}
			}
			if unsetB {
				if unsetMax {
					return -1
				} else {
					return 1
				}
			}

			// If values are of different types, just compare
			// the string representation of the type
			if !zeek.SameType(a.Type, b.Type) {
				return bytes.Compare([]byte(a.Type.String()), []byte(b.Type.String()))
			}

			sf, ok := sorters[a.Type]
			if !ok {
				sf = lookupSorter(a.Type)
				sorters[a.Type] = sf
			}

			v := sf(a.Body, b.Body)
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

func lookupSorter(typ zeek.Type) comparefn {
	switch typ {
	default:
		return func(a, b []byte) int {
			return bytes.Compare(a, b)
		}
	case zeek.TypeBool:
		return func(a, b []byte) int {
			va, err := zeek.DecodeBool(a)
			if err != nil {
				return -1
			}
			vb, err := zeek.DecodeBool(b)
			if err != nil {
				return 1
			}
			if va == vb {
				return 0
			}
			if va {
				return 1
			}
			return -1
		}

	case zeek.TypeString, zeek.TypeEnum:
		return func(a, b []byte) int {
			return bytes.Compare(a, b)
		}

	//XXX note zeek port type can have "/tcp" etc suffix according
	// to docs but we've only encountered ints in data files.
	// need to fix this.  XXX also we should break this sorts
	// into the different types.
	case zeek.TypeInt:
		return func(a, b []byte) int {
			va, err := zeek.DecodeInt(a)
			if err != nil {
				return -1
			}
			vb, err := zeek.DecodeInt(b)
			if err != nil {
				return 1
			}
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}

	case zeek.TypeCount:
		return func(a, b []byte) int {
			va, err := zeek.DecodeCount(a)
			if err != nil {
				return -1
			}
			vb, err := zeek.DecodeCount(b)
			if err != nil {
				return 1
			}
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}
	case zeek.TypePort:
		return func(a, b []byte) int {
			va, err := zeek.DecodePort(a)
			if err != nil {
				return -1
			}
			vb, err := zeek.DecodePort(b)
			if err != nil {
				return 1
			}
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}

	case zeek.TypeDouble:
		return func(a, b []byte) int {
			va, err := zeek.DecodeDouble(a)
			if err != nil {
				return -1
			}
			vb, err := zeek.DecodeDouble(b)
			if err != nil {
				return 1
			}
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}

	case zeek.TypeTime, zeek.TypeInterval:
		return func(a, b []byte) int {
			va, err := zeek.DecodeTime(a)
			if err != nil {
				return -1
			}
			vb, err := zeek.DecodeTime(b)
			if err != nil {
				return 1
			}
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}

	case zeek.TypeAddr:
		return func(a, b []byte) int {
			va, err := zeek.DecodeAddr(a)
			if err != nil {
				return -1
			}
			vb, err := zeek.DecodeAddr(b)
			if err != nil {
				return 1
			}
			return bytes.Compare(va.To16(), vb.To16())
		}
	}
}

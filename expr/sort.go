package expr

import (
	"bytes"
	"sort"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr/coerce"
	"github.com/brimdata/zed/zcode"
)

type CompareFn func(a *zed.Value, b *zed.Value) int

// NewCompareFn creates a function that compares two values a and b according to
// nullsMax and exprs.  To compare a and b, it iterates over the elements e of
// exprs, stopping when e(a)!=e(b).  The handling of missing and null
// (collectively refered to as "null") values is governed by nullsMax.  If
// nullsMax is true, a null value is considered larger than any non-null value,
// and vice versa.
func NewCompareFn(nullsMax bool, exprs ...Evaluator) CompareFn {
	return NewComparator(nullsMax, false, exprs...).WithMissingAsNull().Compare
}

func NewValueCompareFn(nullsMax bool) CompareFn {
	return NewComparator(nullsMax, false, &This{}).Compare
}

type Comparator struct {
	exprs    []Evaluator
	nullsMax bool
	reverse  bool

	comparefns map[zed.Type]comparefn
	ectx       Context
	pair       coerce.Pair
}

type comparefn func(a, b zcode.Bytes) int

// NewComparator returns a zed.Value comparator for exprs according to nullsMax
// and reverse.  To compare values a and b, it iterates over the elements e of
// exprs, stopping when e(a)!=e(b).  nullsMax determines whether a null value
// compares larger (if true) or smaller (if false) than a non-null value.
// reverse reverses the sense of comparisons.
func NewComparator(nullsMax, reverse bool, exprs ...Evaluator) *Comparator {
	return &Comparator{
		exprs:      append([]Evaluator{}, exprs...),
		nullsMax:   nullsMax,
		reverse:    reverse,
		comparefns: make(map[zed.Type]comparefn),
		ectx:       NewContext(),
	}
}

// WithMissingAsNull returns the receiver after modifying it to treat missing
// values as the null value in comparisons.
func (c *Comparator) WithMissingAsNull() *Comparator {
	for i, k := range c.exprs {
		c.exprs[i] = &missingAsNull{k}
	}
	return c
}

type missingAsNull struct{ Evaluator }

func (m *missingAsNull) Eval(ectx Context, val *zed.Value) *zed.Value {
	val = m.Evaluator.Eval(ectx, val)
	if val.IsMissing() {
		return zed.Null
	}
	return val
}

// Compare returns an interger comparing two values according to the receiver's
// configuration.  The result will be 0 if a==b, -1 if a < b, and +1 if a > b.
func (c *Comparator) Compare(a, b *zed.Value) int {
	if c.reverse {
		a, b = b, a
	}
	for _, k := range c.exprs {
		aval := k.Eval(c.ectx, a)
		bval := k.Eval(c.ectx, b)
		if v := compareValues(aval, bval, c.comparefns, &c.pair, c.nullsMax); v != 0 {
			return v
		}
	}
	return 0
}

func compareValues(a, b *zed.Value, comparefns map[zed.Type]comparefn, pair *coerce.Pair, nullsMax bool) int {
	// Handle nulls according to nullsMax
	nullA := a.IsNull()
	nullB := b.IsNull()
	if nullA && nullB {
		return 0
	}
	if nullA {
		if nullsMax {
			return 1
		} else {
			return -1
		}
	}
	if nullB {
		if nullsMax {
			return -1
		} else {
			return 1
		}
	}

	typ := a.Type
	abytes, bbytes := a.Bytes, b.Bytes
	if a.Type.ID() != b.Type.ID() {
		id, err := pair.Coerce(a, b)
		typ = zed.LookupPrimitiveByID(id)
		if err != nil || typ == nil {
			// If values cannot be coerced, just compare the native
			// representation of the type.
			// XXX This is heavyweight and should probably just compare
			// the zcode.Bytes.  See issue #2354.
			return bytes.Compare([]byte(a.Type.String()), []byte(b.Type.String()))
		}
		abytes, bbytes = pair.A, pair.B
	}

	cfn, ok := comparefns[typ]
	if !ok {
		cfn = LookupCompare(typ)
		comparefns[typ] = cfn
	}

	return cfn(abytes, bbytes)
}

// SortStable performs a stable sort on the provided records.
func SortStable(records []zed.Value, compare CompareFn) {
	slice := &RecordSlice{records, compare}
	sort.Stable(slice)
}

type RecordSlice struct {
	vals    []zed.Value
	compare CompareFn
}

func NewRecordSlice(compare CompareFn) *RecordSlice {
	return &RecordSlice{compare: compare}
}

// Swap implements sort.Interface for *Record slices.
func (r *RecordSlice) Len() int { return len(r.vals) }

// Swap implements sort.Interface for *Record slices.
func (r *RecordSlice) Swap(i, j int) { r.vals[i], r.vals[j] = r.vals[j], r.vals[i] }

// Less implements sort.Interface for *Record slices.
func (r *RecordSlice) Less(i, j int) bool {
	return r.compare(&r.vals[i], &r.vals[j]) < 0
}

// Push adds x as element Len(). Implements heap.Interface.
func (r *RecordSlice) Push(rec interface{}) {
	r.vals = append(r.vals, *rec.(*zed.Value))
}

// Pop removes the first element in the array. Implements heap.Interface.
func (r *RecordSlice) Pop() interface{} {
	rec := r.vals[len(r.vals)-1]
	r.vals = r.vals[:len(r.vals)-1]
	return &rec
}

// Index returns the ith record.
func (r *RecordSlice) Index(i int) *zed.Value {
	return &r.vals[i]
}

func LookupCompare(typ zed.Type) comparefn {
	// XXX record support easy to add here if we moved the creation of the
	// field resolvers into this package.
	if innerType := zed.InnerType(typ); innerType != nil {
		return func(a, b zcode.Bytes) int {
			compare := LookupCompare(innerType)
			ia := a.Iter()
			ib := b.Iter()
			for {
				if ia.Done() {
					if ib.Done() {
						return 0
					}
					return -1
				}
				if ib.Done() {
					return 1
				}
				if v := compare(ia.Next(), ib.Next()); v != 0 {
					return v
				}
			}
		}
	}
	switch typ.ID() {
	case zed.IDBool:
		return func(a, b zcode.Bytes) int {
			va, vb := zed.DecodeBool(a), zed.DecodeBool(b)
			if va == vb {
				return 0
			}
			if va {
				return 1
			}
			return -1
		}

	case zed.IDString:
		return func(a, b zcode.Bytes) int {
			return bytes.Compare(a, b)
		}

	case zed.IDInt16, zed.IDInt32, zed.IDInt64:
		return func(a, b zcode.Bytes) int {
			va, vb := zed.DecodeInt(a), zed.DecodeInt(b)
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}

	case zed.IDUint16, zed.IDUint32, zed.IDUint64:
		return func(a, b zcode.Bytes) int {
			va, vb := zed.DecodeUint(a), zed.DecodeUint(b)
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}

	case zed.IDFloat32, zed.IDFloat64:
		return func(a, b zcode.Bytes) int {
			va, vb := zed.DecodeFloat(a), zed.DecodeFloat(b)
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}

	case zed.IDTime:
		return func(a, b zcode.Bytes) int {
			va, vb := zed.DecodeTime(a), zed.DecodeTime(b)
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}

	case zed.IDDuration:
		return func(a, b zcode.Bytes) int {
			va, vb := zed.DecodeDuration(a), zed.DecodeDuration(b)
			if va < vb {
				return -1
			} else if va > vb {
				return 1
			}
			return 0
		}

	case zed.IDIP:
		return func(a, b zcode.Bytes) int {
			va, vb := zed.DecodeIP(a), zed.DecodeIP(b)
			return va.Compare(vb)
		}

	default:
		return func(a, b zcode.Bytes) int {
			return bytes.Compare(a, b)
		}
	}
}

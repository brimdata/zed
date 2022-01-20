package expr

import (
	"bytes"
	"sort"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr/coerce"
	"github.com/brimdata/zed/zcode"
)

type CompareFn func(a *zed.Value, b *zed.Value) int
type KeyCompareFn func(Context, *zed.Value) int

// Internal function that compares two values of compatible types.
type comparefn func(a, b zcode.Bytes) int

// NewCompareFn creates a function that compares a pair of Records
// based on the provided ordered list of fields.
// The returned function uses the same return conventions as standard
// routines such as bytes.Compare() and strings.Compare(), so it may
// be used with packages such as heap and sort.
// The handling of records in which a comparison field is null or not
// present (collectively refered to as fields in which the value is "null")
// is governed by the nullsMax parameter.  If this parameter is true,
// a record with a null value is considered larger than a record with any
// other value, and vice versa.
func NewCompareFn(nullsMax bool, fields ...Evaluator) CompareFn {
	var pair coerce.Pair
	comparefns := make(map[zed.Type]comparefn)
	ectx := NewContext() //XXX should be smarter about this... pass it in?
	return func(ra *zed.Value, rb *zed.Value) int {
		for _, resolver := range fields {
			a := resolver.Eval(ectx, ra)
			if a.IsMissing() {
				// Treat missing values as null so nulls-first/last
				// works for these.
				a = zed.Null
			}
			// XXX We should compute a vector of sort keys then
			// sort the pointers and then generate the batches
			// on demand from the sorted pointers.  And we should
			// special case this for native machine-word keys.
			// i.e., we sort {key,*zed.Value} then build the new
			// batches from the sorted pointers.
			a = a.Copy()

			b := resolver.Eval(ectx, rb)
			if b.IsMissing() {
				b = zed.Null
			}
			v := compareValues(a, b, comparefns, &pair, nullsMax)
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

func NewValueCompareFn(nullsMax bool) CompareFn {
	var pair coerce.Pair
	comparefns := make(map[zed.Type]comparefn)
	return func(a, b *zed.Value) int {
		return compareValues(a, b, comparefns, &pair, nullsMax)
	}
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

func NewKeyCompareFn(zctx *zed.Context, key *zed.Value) (KeyCompareFn, error) {
	comparefns := make(map[zed.Type]comparefn)
	var accessors []Evaluator
	var keyval []zed.Value
	for it := key.FieldIter(); !it.Done(); {
		name, val, err := it.Next()
		if err != nil {
			return nil, err
		}
		// We got a null  value, so all remaining values in the key
		// must be null.
		if val.IsNull() {
			break
		}
		keyval = append(keyval, val)
		accessors = append(accessors, NewDottedExpr(zctx, name))
	}
	return func(ectx Context, this *zed.Value) int {
		for k, access := range accessors {
			// XXX error
			a := access.Eval(ectx, this)
			if a.IsNull() {
				// we know the key value is not null
				return -1
			}
			//XXX I think we can take this out now that values
			// are all allocated in the context... (need to hit funcs)
			a = a.Copy()

			b := keyval[k]
			// If the type of a field in the comparison record does
			// not match the type of the key, behavior is undefined.
			if a.Type.ID() != b.Type.ID() {
				return -1
			}
			//XXX comparefns should be a slice indexed by ID
			cfn, ok := comparefns[a.Type]
			if !ok {
				cfn = LookupCompare(a.Type)
				comparefns[a.Type] = cfn
			}
			v := cfn(a.Bytes, b.Bytes)
			// If the fields don't match, then return the sense of
			// the mismatch.  Otherwise, we continue on
			// in the loop to the secondary key, etc.
			if v != 0 {
				return v
			}
		}
		// All the keys matched with equality.
		return 0
	}, nil
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
				va, container := ia.Next()
				if container {
					return -1
				}

				vb, container := ib.Next()
				if container {
					return 1
				}
				if v := compare(va, vb); v != 0 {
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
			return bytes.Compare(va.To16(), vb.To16())
		}

	default:
		return func(a, b zcode.Bytes) int {
			return bytes.Compare(a, b)
		}
	}
}

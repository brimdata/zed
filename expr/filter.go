package expr

import (
	"errors"
	"strings"

	"github.com/brimdata/zed"
	astzed "github.com/brimdata/zed/compiler/ast/zed"
	"github.com/brimdata/zed/pkg/byteconv"
	"github.com/brimdata/zed/pkg/stringsearch"
	"github.com/brimdata/zed/zcode"
)

type Filter func(Context, *zed.Value) bool

func LogicalAnd(left, right Filter) Filter {
	return func(ctx Context, val *zed.Value) bool {
		return left(ctx, val) && right(ctx, val)
	}
}

func LogicalOr(left, right Filter) Filter {
	return func(ctx Context, val *zed.Value) bool {
		return left(ctx, val) || right(ctx, val)
	}
}

func LogicalNot(expr Filter) Filter {
	return func(ctx Context, val *zed.Value) bool {
		return !expr(ctx, val)
	}
}

func Apply(e Evaluator, pred Boolean) Filter {
	return func(ctx Context, val *zed.Value) bool {
		v := e.Eval(ctx, val)
		if v.IsError() {
			// There's no wy to propagate errors in a filter
			// because the predicate never lands anywhere.
			// It could make sense to simply propagate errors
			// out of the filter when the predicate produces them
			// as you could always apply a Zed operator to ignore them.
			// e.g., ignore_err(a / b > 10) could turn divide-by-zero errors
			// into missing.  We would need to change the type
			// signature of Filter to deal with this.
			// Also, we should wrap the error when we have structured
			// errors so the user knows the error came from the filter.
			return false
		}
		return pred(v)
	}
}

func EvalAny(eval Boolean, recursive bool) Filter {
	if !recursive {
		return func(ctx Context, val *zed.Value) bool {
			it := val.Bytes.Iter()
			for _, c := range val.Columns() {
				val, _, err := it.Next()
				if err != nil {
					return false
				}
				//XXX put value stash in closure above...
				// as long as there is one per thread?
				if eval(&zed.Value{c.Type, val}) {
					return true
				}
			}
			return false
		}
	}
	//XXX
	var fn func(v zcode.Bytes, cols []zed.Column) bool
	fn = func(v zcode.Bytes, cols []zed.Column) bool {
		it := v.Iter()
		for _, c := range cols {
			val, _, err := it.Next()
			if err != nil {
				return false
			}
			recType, isRecord := c.Type.(*zed.TypeRecord)
			if isRecord && fn(val, recType.Columns) {
				return true
				//XXX &val
			} else if !isRecord && eval(&zed.Value{c.Type, val}) {
				return true
			}
		}
		return false
	}
	return func(_ Context, val *zed.Value) bool {
		return fn(val.Bytes, val.Columns())
	}
}

// stringSearch is like strings.Contains() but with case-insensitive
// comparison.
func stringSearch(a, b string) bool {
	alen := len(a)
	blen := len(b)

	if blen > alen {
		return false
	}

	end := alen - blen + 1
	i := 0
	for i < end {
		if strings.EqualFold(a[i:i+blen], b) {
			return true
		}
		i++
	}
	return false
}

var errMatch = errors.New("match")

// SearchRecordOther creates a filter that searches zng records for the
// given value, which must be of a type other than (b)string.  The filter
// matches a record that contains this value either as the value of any
// field or inside any set or array.  It also matches a record if the string
// representaton of the search value appears inside inside any string-valued
// field (or inside any element of a set or array of strings).
func SearchRecordOther(searchtext string, searchval astzed.Primitive) (Filter, error) {
	typedCompare, err := Comparison("=", searchval)
	if err != nil {
		return nil, err
	}
	return func(_ Context, val *zed.Value) bool {
		return errMatch == val.Walk(func(typ zed.Type, body zcode.Bytes) error {
			if zed.IsStringy(typ.ID()) {
				if stringSearch(byteconv.UnsafeString(body), searchtext) {
					return errMatch
				}
				//XXX &val
			} else if typedCompare(&zed.Value{Type: typ, Bytes: body}) {
				return errMatch
			}
			return nil
		})
	}, nil

}

// SearchRecordString handles the special case of string searching -- it
// matches both field names and values.
func SearchRecordString(term string) Filter {
	fieldNameCheck := make(map[zed.Type]bool)
	var nameIter stringsearch.FieldNameIter
	return func(_ Context, val *zed.Value) bool {
		// Memoize the result of a search across the names in the
		// record columns for each unique record type.
		match, ok := fieldNameCheck[val.Type]
		if !ok {
			nameIter.Init(zed.TypeRecordOf(val.Type))
			for !nameIter.Done() {
				if stringSearch(byteconv.UnsafeString(nameIter.Next()), term) {
					match = true
					break
				}
			}
			fieldNameCheck[val.Type] = match
		}
		if match {
			return true
		}
		return errMatch == val.Walk(func(typ zed.Type, body zcode.Bytes) error {
			if zed.IsStringy(typ.ID()) &&
				stringSearch(byteconv.UnsafeString(body), term) {
				return errMatch
			}
			return nil
		})
	}
}

type FilterEvaluator Filter

func (f FilterEvaluator) Eval(ctx Context, this *zed.Value) *zed.Value {
	if f(ctx, this) {
		return zed.True
	}
	return zed.False
}

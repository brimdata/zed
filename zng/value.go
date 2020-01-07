package zng

import (
	"errors"
	"fmt"

	"github.com/mccanne/zq/ast"
	"github.com/mccanne/zq/zcode"
)

var (
	ErrNotNumber       = errors.New("not a number")
	ErrTypeSyntax      = errors.New("syntax error parsing type string")
	ErrDuplicateFields = errors.New("duplicate fields in record type")
)

type TypedEncoding struct {
	Type Type
	Body zcode.Bytes
}

// A Predicate is a function that takes a Type and a byte slice, parses the
// byte slice according the Type, and returns a boolean result based on the
// typed value.  For example, each Value has a Comparison method that returns
// a Predicate for comparing byte slices to that value.
type Predicate func(TypedEncoding) bool

// Value is the interface that all zeek values implement.
type Value interface {
	// Return the string representation of the value, e.g., what appears
	// in a zeek log file.
	String() string
	// Return the Type of this value.
	Type() Type
	// Return a predicate for comparing this value to one more typed
	// byte slices by calling the predicate function with a Type and
	// a byte slice.  Operand is one of "eql", "neql", "lt", "lte",
	// "gt", "gte".  See the comments of the various implementation
	// of this method as some types limit the operand to equality and
	// the various types handle coercion in different ways.
	Comparison(operator string) (Predicate, error)
	// Coerce tries to convert this value to an equal value of a different
	// type.  For example, calling Coerce(TypeDouble) on a value that is
	// an Int{100} will return a Double{100.}.  If the coercion cannot be
	// performed such that v.Coerce(t1).Coerce(v.Type).String() == v.String(),
	// then nil is returned.
	Coerce(Type) Value
	// If this value is a container (set or vector), return an array of
	// the contained Values and true.  If this value is not a container,
	// return an empty list and false.
	Elements() ([]Value, bool)
	// Encode appends the zval representation of this value to the passed in
	// argument and returns the resulting zcode.Bytes (which may or may not
	// be the same underlying buffer, as with append(), depending on its capacity)
	Encode(zcode.Bytes) zcode.Bytes
}

// Parse translates an ast.TypedValue into a zeek.Value.
func Parse(v ast.TypedValue) (Value, error) {
	typeMapMutex.RLock()
	t, ok := typeMap[v.Type]
	typeMapMutex.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unsupported type %s in ast TypedValue", v.Type)
	}
	if t == TypePattern || t == TypeString {
		return t.New(Unescape([]byte(v.Value)))
	}
	zv, err := t.Parse([]byte(v.Value))
	if err != nil {
		return nil, err
	}
	return t.New(zv)
}

func parseContainer(containerType Type, elementType Type, b []byte) ([]Value, error) {
	// We start out with a pointer instead of nil so that empty sets and vectors
	// are properly encoded etc., e.g., by json.Marshal.
	vals := make([]Value, 0)
	for it := zcode.Iter(b); !it.Done(); {
		val, _, err := it.Next()
		if err != nil {
			return nil, fmt.Errorf("parsing %s element %q: %w", containerType.String(), val, err)
		}
		v, err := elementType.New(val)
		if err != nil {
			return nil, fmt.Errorf("parsing %s element %q: %w", containerType.String(), val, err)
		}
		vals = append(vals, v)
	}
	return vals, nil
}

func IsContainer(v Value) bool {
	_, ok := v.Elements()
	return ok
}

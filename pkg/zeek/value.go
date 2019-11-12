package zeek

import (
	"errors"
	"fmt"

	"github.com/mccanne/zq/ast"
	"github.com/mccanne/zq/pkg/zval"
)

var (
	ErrNotNumber  = errors.New("not a number")
	ErrTypeSyntax = errors.New("syntax error parsing type string")
)

// A Predicate is a function that takes a Type and a byte slice, parses the
// byte slice according the Type, and returns a boolean result based on the
// typed value.  For example, each Value has a Comparison method that returns
// a Predicate for comparing byte slices to that value.
type Predicate func(typ Type, val []byte) bool

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
	// XXX we will add Marshal and Unmarshal when we move zeek.Value
	// into the ast

	// If this value is a container (set or vector), return an array of
	// the contained Values and true.  If this value is not a container,
	// return an empty list and false.
	Elements() ([]Value, bool)
}

// Parse translates an ast.TypedValue into a zeek.Value.
// XXX at some point, we will move zeek.Value into the ast and make
// this automatic.  At that point, ast will depend on package zeek
// and we'll need to get rid of this.
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
	return t.New([]byte(v.Value))
}

func parseContainer(containerType Type, elementType Type, b []byte) ([]Value, error) {
	var vals []Value
	for it := zval.Iter(b); !it.Done(); {
		val, err := it.Next()
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

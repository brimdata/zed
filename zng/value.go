package zng

import (
	"encoding/json"
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

type Value struct {
	Type  Type
	Bytes zcode.Bytes
}

// NewValue creates a Value with the given type and value described
// as simple strings.
func NewValue(typ, val string) (Value, error) {
	t, err := LookupType(typ)
	if err != nil {
		return Value{}, err
	}
	zv, err := t.Parse([]byte(val))
	if err != nil {
		return Value{}, err
	}
	return Value{t, zv}, nil
}

// Parse translates an Literal into a Value.
func Parse(v ast.Literal) (Value, error) {
	typeMapMutex.RLock()
	t, ok := typeMap[v.Type]
	typeMapMutex.RUnlock()
	if !ok {
		return Value{}, fmt.Errorf("unsupported type %s in ast.Literal", v.Type)
	}
	zv, err := t.Parse([]byte(v.Value))
	if err != nil {
		return Value{}, err
	}
	return Value{t, zv}, nil
}

//XXX b should be zcode.Bytes
func parseContainer(containerType Type, elementType Type, b []byte) ([]Value, error) {
	// We start out with a pointer instead of nil so that empty sets and vectors
	// are properly encoded etc., e.g., by json.Marshal.
	vals := make([]Value, 0)
	for it := zcode.Iter(b); !it.Done(); {
		zv, _, err := it.Next()
		if err != nil {
			return nil, fmt.Errorf("parsing %s element %q: %w", containerType.String(), zv, err)
		}
		vals = append(vals, Value{elementType, zv})
	}
	return vals, nil
}

func (v Value) IsContainer() bool {
	return IsContainerType(v.Type)
}

func (v Value) MarshalJSON() ([]byte, error) {
	if v.Bytes == nil {
		return json.Marshal(nil)
	}
	object, err := v.Type.Marshal(v.Bytes)
	if err != nil {
		return nil, err
	}
	return json.Marshal(object)
}

func badZng(err error, t Type, zv zcode.Bytes) string {
	return fmt.Sprintf("<ZNG-ERR type %s [%s]: %s>", t, zv, err)
}

// String implements the fmt.Stringer interfae and returns the string representation
// of the value in accordance with the ZNG spec.  XXX currently we're not doing
// escaping here.
func (v Value) String() string {
	if v.Bytes == nil {
		return "-"
	}
	return v.Type.StringOf(v.Bytes)
}

// Format tranforms a zval encoding with its type encoding to a
// a human-readable (and zng text-compliant) string format
// encoded as a byte slice.
//XXX this could be more efficient
func (v Value) Format() []byte {
	return []byte(Escape([]byte(v.String())))
}

// Encode appends the BZNG representation of this value to the passed in
// argument and returns the resulting zcode.Bytes (which may or may not
// be the same underlying buffer, as with append(), depending on its capacity)
func (v Value) Encode(dst zcode.Bytes) zcode.Bytes {
	if IsContainerType(v.Type) {
		return zcode.AppendContainer(dst, v.Bytes)
	}
	return zcode.AppendPrimitive(dst, v.Bytes)
}

func (v Value) Iter() zcode.Iter {
	return zcode.Iter(v.Bytes)
}

// If the passed-in element is a vector, attempt to get the idx'th
// element, and return its type and raw representation.  Returns an
// error if the passed-in element is not a vector or if idx is
// outside the vector bounds.
func (v Value) VectorIndex(idx int64) (Value, error) {
	vec, ok := v.Type.(*TypeVector)
	if !ok {
		return Value{}, ErrNotVector
	}
	if idx < 0 {
		return Value{}, ErrIndex
	}
	for i, it := 0, v.Iter(); !it.Done(); i++ {
		zv, _, err := it.Next()
		if err != nil {
			return Value{}, err
		}
		if i == int(idx) {
			return Value{vec.typ, zv}, nil
		}
	}
	return Value{}, ErrIndex
}

// Elements returns an array of Values for the given container type.
// Returns an error if the element is not a vector or set.
func (v Value) Elements() ([]Value, error) {
	innerType := InnerType(v.Type)
	if innerType == nil {
		return nil, ErrNotContainer
	}
	var elements []Value
	for it := v.Iter(); !it.Done(); {
		zv, _, err := it.Next()
		if err != nil {
			return nil, err
		}
		elements = append(elements, Value{innerType, zv})
	}
	return elements, nil
}

func (v Value) ContainerLength() (int, error) {
	switch v.Type.(type) {
	case *TypeSet, *TypeVector:
		if v.Bytes == nil {
			return -1, ErrLenUnset
		}
		var n int
		for it := v.Iter(); !it.Done(); {
			if _, _, err := it.Next(); err != nil {
				return -1, err
			}
			n++
		}
		return n, nil
	default:
		return -1, ErrNotContainer
	}
}

func (v Value) IsNil() bool {
	return v.Bytes == nil && v.Type == nil
}

// IsUnset returns true iff v is an unset value.  Unset values are represented
// with a zero-valued Value.  A zero-valued value that is not unset is represented
// by a non-nil slice for Bytes of zero length.
func (v Value) IsUnset() bool {
	return v.Bytes == nil && v.Type != nil
}

func (v Value) IsUnsetOrNil() bool {
	return v.Bytes == nil
}

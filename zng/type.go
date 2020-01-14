// Package zng implements a data typing system based on the zeek type system.
// All zeek types are defined here and implement the Type interface while instances
// of values implement the Value interface.  All values conform to exactly one type.
// The package provides a fast-path for comparing a value to a byte slice
// without having to create a zeek value from the byte slice.  To exploit this,
// all values include a Comparison method that returns a Predicate function that
// takes a byte slice and a Type and returns a boolean indicating whether the
// the byte slice with the indicated Type matches the value.  The package also
// provides mechanism for coercing values in well-defined and natural ways.
package zng

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/mccanne/zq/zcode"
)

var (
	ErrUnset        = errors.New("value is unset")
	ErrLenUnset     = errors.New("len(unset) is undefined")
	ErrNotContainer = errors.New("argument to len() is not a container")
	ErrNotVector    = errors.New("cannot index a non-vector")
	ErrIndex        = errors.New("vector index out of bounds")
)

// A Type is an interface presented by a zeek type.
// Types can be used to infer type compatibility and create new values
// of the underlying type.
type Type interface {
	String() string
	StringOf(zcode.Bytes) string
	Marshal(zcode.Bytes) (interface{}, error)
	// Parse transforms a string represenation of the type to its zval
	// encoding.  The string input is provided as a byte slice for efficiency
	// given the common use cases in the system.
	Parse([]byte) (zcode.Bytes, error)
}

var (
	TypeBool     = &TypeOfBool{}
	TypeCount    = &TypeOfCount{}
	TypeInt      = &TypeOfInt{}
	TypeDouble   = &TypeOfDouble{}
	TypeTime     = &TypeOfTime{}
	TypeInterval = &TypeOfInterval{}
	TypeString   = &TypeOfString{}
	TypePort     = &TypeOfPort{}
	TypeAddr     = &TypeOfAddr{}
	TypeSubnet   = &TypeOfSubnet{}
	TypeEnum     = &TypeOfEnum{}
)

var typeMapMutex sync.RWMutex
var typeMap = map[string]Type{
	"bool":     TypeBool,
	"count":    TypeCount,
	"int":      TypeInt,
	"double":   TypeDouble,
	"time":     TypeTime,
	"interval": TypeInterval,
	"string":   TypeString,
	"port":     TypePort,
	"addr":     TypeAddr,
	"subnet":   TypeSubnet,
	"enum":     TypeEnum,
}

// SameType returns true if the two types are equal in that each interface
// points to the same underlying type object.  Because the zeek library
// creates each unique type only once, this pointer comparison works.  If types
// are created outside of the zeek package, then SameType will not work in general
// for them.
func SameType(t1, t2 Type) bool {
	return t1 == t2
}

// addType adds a type to the type lookup map.
func addType(t Type) Type {
	typeMapMutex.Lock()
	defer typeMapMutex.Unlock()
	key := t.String()
	old, ok := typeMap[key]
	if ok {
		t = old
	} else {
		typeMap[key] = t
	}
	return t
}

func isIdChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '.'
}

func parseWord(in string) (string, string) {
	in = strings.TrimSpace(in)
	var off int
	for ; off < len(in); off++ {
		if !isIdChar(in[off]) {
			break
		}
	}
	if off == 0 {
		return "", ""
	}
	return in[off:], in[:off]
}

// LookupType returns the Type indicated by the zeek type string.  The type string
// may be a simple type like int, double, time, etc or it may be a set
// or a vector, which are recusively composed of other types.  The set and vector
// type definitions are encoded in the same fashion as zeek stores them as type field
// in a zeek file header.  Each unique compound type object is created once and
// interned so that pointer comparison can be used to determine type equality.
func LookupType(in string) (Type, error) {
	//XXX check if rest has junk and flag an error?
	_, typ, err := parseType(in)
	return typ, err
}

// LookupVectorType returns the VectorType for the provided innerType.
func LookupVectorType(innerType Type) Type {
	return addType(&TypeVector{typ: innerType})
}

func parseType(in string) (string, Type, error) {
	typeMapMutex.RLock()
	t, ok := typeMap[strings.TrimSpace(in)]
	typeMapMutex.RUnlock()
	if ok {
		return "", t, nil
	}
	rest, word := parseWord(in)
	if word == "" {
		return "", nil, fmt.Errorf("unknown type: %s", in)
	}
	typeMapMutex.RLock()
	t, ok = typeMap[word]
	typeMapMutex.RUnlock()
	if ok {
		return rest, t, nil
	}
	switch word {
	case "set":
		rest, t, err := parseSetTypeBody(rest)
		if err != nil {
			return "", nil, err
		}
		return rest, addType(t), nil
	case "vector":
		rest, t, err := parseVectorTypeBody(rest)
		if err != nil {
			return "", nil, err
		}
		return rest, addType(t), nil
	case "record":
		rest, t, err := parseRecordTypeBody(rest)
		if err != nil {
			return "", nil, err
		}
		return rest, addType(t), nil
	}
	return "", nil, fmt.Errorf("unknown type: %s", word)
}

// Utilities shared by compound types (ie, set and vector)

// InnerType returns the element type for set and vector types
// or nil if the type is not a set or vector.
func InnerType(typ Type) Type {
	switch typ := typ.(type) {
	case *TypeSet:
		return typ.innerType
	case *TypeVector:
		return typ.typ
	default:
		return nil
	}
}

// ContainedType returns the inner type for set and vector types in the first
// return value and the columns of its of type for record types in the second
// return value.  ContainedType returns nil for both return values if the
// type is not a set, vector, or record.
func ContainedType(typ Type) (Type, []Column) {
	switch typ := typ.(type) {
	case *TypeSet:
		return typ.innerType, nil
	case *TypeVector:
		return typ.typ, nil
	case *TypeRecord:
		return nil, typ.Columns
	default:
		return nil, nil
	}
}

func IsContainerType(typ Type) bool {
	switch typ.(type) {
	case *TypeSet, *TypeVector, *TypeRecord:
		return true
	default:
		return false
	}
}

func trimInnerTypes(typ string, raw string) string {
	// XXX handle white space, "set [..."... ?
	innerTypes := strings.TrimPrefix(raw, typ+"[")
	innerTypes = strings.TrimSuffix(innerTypes, "]")
	return innerTypes
}

// LookupTypeRecord returns a zeek.TypeRecord for the indicated columns.  If it
// already exists, the existent interface pointer is returned.  Otherwise,
// it is created and returned.
func LookupTypeRecord(columns []Column) *TypeRecord {
	s := recordString(columns)
	typeMapMutex.RLock()
	t, ok := typeMap[s]
	typeMapMutex.RUnlock()
	if ok {
		return t.(*TypeRecord)
	}
	typeMapMutex.Lock()
	defer typeMapMutex.Unlock()
	t, ok = typeMap[s]
	if ok {
		return t.(*TypeRecord)
	}
	// Make a private copy of the columns to maintain the invariant
	// that types are immutable and the columns can be retrieved from
	// the type system and traversed without any data races.
	private := make([]Column, len(columns))
	for k, p := range columns {
		private[k] = p
	}
	rec := &TypeRecord{Columns: private, Key: s}
	typeMap[s] = rec
	return rec
}

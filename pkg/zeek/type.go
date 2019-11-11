// Package zeek implements a data typing system based on the zeek type system.
// All zeek types are defined here and implement the Type interface while instances
// of values implement the Value interface.  All values conform to exactly one type.
// The package provides a fast-path for comparing a value to a byte slice
// without having to create a zeek value from the byte slice.  To exploit this,
// all values include a Comparison method that returns a Predicate function that
// takes a byte slice and a Type and returns a boolean indicating whether the
// the byte slice with the indicated Type matches the value.  The package also
// provides mechanism for coercing values in well-defined and natural ways.
package zeek

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/mccanne/zq/pkg/zval"
)

var (
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
	// New returns a Value of this Type by parsing the data in the bye slice
	// and interpreting it as the native value of the zeek Value.
	New([]byte) (Value, error)
	// Format returns a native value as an empty interface by parsing the
	// data in the byte slice as an instance of this Type.  This allows
	// the creation of native values from a Type without having to allocate
	// a zeek Value.
	Format([]byte) (interface{}, error)
}

var (
	TypeBool     = &TypeOfBool{}
	TypeCount    = &TypeOfCount{}
	TypeInt      = &TypeOfInt{}
	TypeDouble   = &TypeOfDouble{}
	TypeTime     = &TypeOfTime{}
	TypeInterval = &TypeOfInterval{}
	TypeString   = &TypeOfString{}
	TypePattern  = &TypeOfPattern{}
	TypePort     = &TypeOfPort{}
	TypeAddr     = &TypeOfAddr{}
	TypeSubnet   = &TypeOfSubnet{}
	TypeEnum     = &TypeOfEnum{}
	TypeNone     = &TypeOfNone{}
	TypeUnset    = &TypeOfUnset{}
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
	"pattern":  TypePattern,
	"regexp":   TypePattern, // lookytalk
	"port":     TypePort,
	"addr":     TypeAddr,
	"subnet":   TypeSubnet,
	"enum":     TypeEnum,
	"unset":    TypeUnset, // lookytalk
	"none":     TypeNone,
}

// SameType returns true if the two types are equal in that each interface
// points to the same underlying type object.  Because the zeek library
// creates each unique type only once, this pointer comparison works.  If types
// are created outside of the zeek package, then SameType will not work in general
// for them.
func SameType(t1, t2 Type) bool {
	return t1 == t2
}

// addType adds a type to the type lookup map.  It is possible that there is
// a race here when two threads try to create a new type at the same time,
// so the first one wins.  This way there cannot be types that are the same
// that have different pointers, so SameType will work correctly.
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
		if err == nil {
			return rest, addType(t), nil
		}
	case "vector":
		rest, t, err := parseVectorTypeBody(rest)
		if err == nil {
			return rest, addType(t), nil
		}
	case "record":
		rest, t, err := parseRecordTypeBody(rest)
		if err == nil {
			return rest, addType(t), nil
		}
	}
	return "", nil, fmt.Errorf("unknown type: %s", word)
}

// Utilities shared by compound types (ie, set and vector)

// If the passed-in type is a container, ContainedType() returns
// the type of individual elements (and true for the second value).
// Otherwise, returns nil, false.
func ContainedType(typ Type) (Type, bool) {
	switch typ := typ.(type) {
	case *TypeSet:
		return typ.innerType, true
	case *TypeVector:
		return typ.typ, true
	default:
		return nil, false
	}
}

func trimInnerTypes(typ string, raw string) string {
	// XXX handle white space, "set [..."... ?
	innerTypes := strings.TrimPrefix(raw, typ+"[")
	innerTypes = strings.TrimSuffix(innerTypes, "]")
	return innerTypes
}

// Given a predicate for comparing individual elements, produce a new
// predicate that implements the "in" comparison.  The new predicate looks
// at the type of the value being compared, if it is a set or vector,
// the original predicate is applied to each element.  The new precicate
// returns true iff the predicate matched an element from the collection.
func Contains(compare Predicate) Predicate {
	return func(typ Type, val []byte) bool {
		var elType Type
		switch typ := typ.(type) {
		case *TypeSet:
			elType = typ.innerType
		case *TypeVector:
			elType = typ.typ
		default:
			return false
		}
		for it := zval.Iter(val); !it.Done(); {
			val, err := it.Next()
			if err != nil {
				return false
			}
			if compare(elType, val) {
				return true
			}
		}
		return false
	}
}

func ContainerLength(typ Type, val []byte) (int, error) {
	switch typ.(type) {
	case *TypeSet, *TypeVector:
		if val == nil {
			return -1, ErrLenUnset
		}
		var n int
		for it := zval.Iter(val); !it.Done(); {
			if _, err := it.Next(); err != nil {
				return -1, err
			}
			n++
		}
		return n, nil
	default:
		return -1, ErrNotContainer
	}
}

// If the passed-in element is a vector, attempt to get the idx'th
// element, and return its type and raw representation.  Returns an
// error if the passed-in element is not a vector or if idx is
// outside the vector bounds.
func VectorIndex(typ Type, val []byte, idx int64) (Type, []byte, error) {
	vec, ok := typ.(*TypeVector)
	if !ok {
		return nil, nil, ErrNotVector
	}
	if idx < 0 {
		return nil, nil, ErrIndex
	}
	for i, it := 0, zval.Iter(val); !it.Done(); i++ {
		v, err := it.Next()
		if err != nil {
			return nil, nil, err
		}
		if i == int(idx) {
			return vec.typ, v, nil
		}
	}
	return nil, nil, ErrIndex
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
	rec := &TypeRecord{Columns: columns, Key: s}
	typeMap[s] = rec
	return rec
}

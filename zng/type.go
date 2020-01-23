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
	"strings"

	"github.com/mccanne/zq/zcode"
)

var (
	ErrUnset     = errors.New("value is unset")
	ErrLenUnset  = errors.New("len(unset) is undefined")
	ErrNotVector = errors.New("cannot index a non-vector")
	ErrIndex     = errors.New("vector index out of bounds")
)

// Resolver is an interface for looking up Type objects from the type id.
type Resolver interface {
	//XXX TypeRecord for now
	Lookup(int) *TypeRecord
}

// Context is an interface for looking up TypeRecord objects from a slice of Columns.
type Context interface {
	LookupByColumns([]Column) *TypeRecord
	LookupByName(string) (Type, error)
}

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
	ID() int
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
	TypeNull     = &TypeOfNull{}
)

const (
	IdBool     = 0
	IdByte     = 1
	IdInt16    = 2
	IdUint16   = 3
	IdInt32    = 4
	IdUint32   = 5
	IdInt64    = 6
	IdUint64   = 7
	IdFloat64  = 8
	IdString   = 9
	IdBytes    = 10
	IdBstring  = 11
	IdEnum     = 12
	IdIP       = 13
	IdPort     = 14
	IdNet      = 15
	IdTime     = 16
	IdDuration = 17
	IdNull     = 18
)

func LookupPrimitive(name string) Type {
	switch name {
	case "bool":
		return TypeBool
	case "count":
		return TypeCount
	case "int":
		return TypeInt
	case "double":
		return TypeDouble
	case "time":
		return TypeTime
	case "interval":
		return TypeInterval
	case "string":
		return TypeString
	case "port":
		return TypePort
	case "addr":
		return TypeAddr
	case "subnet":
		return TypeSubnet
	case "enum":
		return TypeEnum
	case "null":
		return TypeNull
	}
	return nil
}

func LookupPrimitiveById(id int) Type {
	switch id {
	case IdBool:
		return TypeBool
	case IdUint64:
		return TypeCount
	case IdInt64:
		return TypeInt
	case IdFloat64:
		return TypeDouble
	case IdTime:
		return TypeTime
	case IdDuration:
		return TypeInterval
	case IdBstring:
		return TypeString
	case IdPort:
		return TypePort
	case IdIP:
		return TypeAddr
	case IdNet:
		return TypeSubnet
	case IdEnum:
		return TypeEnum
	}
	return nil
}

// SameType returns true if the two types are equal in that each interface
// points to the same underlying type object.  Because the zeek library
// creates each unique type only once, this pointer comparison works.  If types
// are created outside of the zeek package, then SameType will not work in general
// for them.
func SameType(t1, t2 Type) bool {
	return t1 == t2
}

// Utilities shared by compound types (ie, set and vector)

// InnerType returns the element type for set and vector types
// or nil if the type is not a set or vector.
func InnerType(typ Type) Type {
	switch typ := typ.(type) {
	case *TypeSet:
		return typ.InnerType
	case *TypeVector:
		return typ.Type
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
		return typ.InnerType, nil
	case *TypeVector:
		return typ.Type, nil
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

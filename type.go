// Package zng implements a data typing system based on the zeek type system.
// All zeek types are defined here and implement the Type interface while instances
// of values implement the Value interface.  All values conform to exactly one type.
// The package provides a fast-path for comparing a value to a byte slice
// without having to create a zeek value from the byte slice.  To exploit this,
// all values include a Comparison method that returns a Predicate function that
// takes a byte slice and a Type and returns a boolean indicating whether the
// the byte slice with the indicated Type matches the value.  The package also
// provides mechanism for coercing values in well-defined and natural ways.
package zed

import (
	"errors"

	"github.com/brimdata/zed/zcode"
)

var (
	ErrNotArray      = errors.New("cannot index a non-array")
	ErrIndex         = errors.New("array index out of bounds")
	ErrUnionSelector = errors.New("union selector out of bounds")
	ErrEnumIndex     = errors.New("enum index out of bounds")
)

// A Type is an interface presented by a zeek type.
// Types can be used to infer type compatibility and create new values
// of the underlying type.
type Type interface {
	// ID returns a unique (per Context) identifier that
	// represents this type.  For an aliased type, this identifier
	// represents the actual underlying type and not the alias itself.
	// Callers that care about the underlying type of a Value for
	// example should prefer to use this instead of using the go
	// .(type) operator on a Type instance.
	ID() int
	String() string
	Format(zv zcode.Bytes) string
}

var (
	TypeUint8    = &TypeOfUint8{}
	TypeUint16   = &TypeOfUint16{}
	TypeUint32   = &TypeOfUint32{}
	TypeUint64   = &TypeOfUint64{}
	TypeInt8     = &TypeOfInt8{}
	TypeInt16    = &TypeOfInt16{}
	TypeInt32    = &TypeOfInt32{}
	TypeInt64    = &TypeOfInt64{}
	TypeDuration = &TypeOfDuration{}
	TypeTime     = &TypeOfTime{}
	// XXX add TypeFloat16
	TypeFloat32 = &TypeOfFloat32{}
	TypeFloat64 = &TypeOfFloat64{}
	// XXX add TypeDecimal
	TypeBool   = &TypeOfBool{}
	TypeBytes  = &TypeOfBytes{}
	TypeString = &TypeOfString{}
	TypeIP     = &TypeOfIP{}
	TypeNet    = &TypeOfNet{}
	TypeType   = &TypeOfType{}
	TypeNull   = &TypeOfNull{}
)

// Primary Type IDs

const (
	IDUint8       = 0
	IDUint16      = 1
	IDUint32      = 2
	IDUint64      = 3
	IDUint128     = 4
	IDUint256     = 5
	IDInt8        = 6
	IDInt16       = 7
	IDInt32       = 8
	IDInt64       = 9
	IDInt128      = 10
	IDInt256      = 11
	IDDuration    = 12
	IDTime        = 13
	IDFloat16     = 14
	IDFloat32     = 15
	IDFloat64     = 16
	IDFloat128    = 17
	IDFloat256    = 18
	IDDecimal32   = 19
	IDDecimal64   = 20
	IDDecimal128  = 21
	IDDecimal256  = 22
	IDBool        = 23
	IDBytes       = 24
	IDString      = 25
	IDIP          = 26
	IDNet         = 27
	IDType        = 28
	IDNull        = 29
	IDTypeComplex = 30
)

// Encodings for complex type values.

const (
	TypeValueRecord  = 30
	TypeValueArray   = 31
	TypeValueSet     = 32
	TypeValueMap     = 33
	TypeValueUnion   = 34
	TypeValueEnum    = 35
	TypeValueError   = 36
	TypeValueNameDef = 37
	TypeValueNameRef = 38
	TypeValueMax     = TypeValueNameRef
)

// True iff the type id is encoded as a zng signed or unsigened integer zcode.Bytes.
func IsInteger(id int) bool {
	return id <= IDInt256
}

// True iff the type id is encoded as a zng signed or unsigned integer zcode.Bytes,
// float32 zcode.Bytes, or float64 zcode.Bytes.
func IsNumber(id int) bool {
	return id <= IDDecimal256
}

// True iff the type id is encoded as a float encoding.
// XXX add IDDecimal here when we implement coercible math with it.
func IsFloat(id int) bool {
	return id >= IDFloat16 && id <= IDFloat256
}

// True iff the type id is encoded as a number encoding and is signed.
func IsSigned(id int) bool {
	return id >= IDInt8 && id <= IDTime
}

func LookupPrimitive(name string) Type {
	switch name {
	case "uint8":
		return TypeUint8
	case "uint16":
		return TypeUint16
	case "uint32":
		return TypeUint32
	case "uint64":
		return TypeUint64
	case "int8":
		return TypeInt8
	case "int16":
		return TypeInt16
	case "int32":
		return TypeInt32
	case "int64":
		return TypeInt64
	case "duration":
		return TypeDuration
	case "time":
		return TypeTime
	case "float32":
		return TypeFloat32
	case "float64":
		return TypeFloat64
	case "bool":
		return TypeBool
	case "bytes":
		return TypeBytes
	case "string":
		return TypeString
	case "ip":
		return TypeIP
	case "net":
		return TypeNet
	case "type":
		return TypeType
	case "null":
		return TypeNull
	}
	return nil
}

func LookupPrimitiveByID(id int) Type {
	if id == 17 {
		// XXX this will be soon removed with Zed formats update
		panic("bstring type is deprecated")
	}
	switch id {
	case IDBool:
		return TypeBool
	case IDInt8:
		return TypeInt8
	case IDUint8:
		return TypeUint8
	case IDInt16:
		return TypeInt16
	case IDUint16:
		return TypeUint16
	case IDInt32:
		return TypeInt32
	case IDUint32:
		return TypeUint32
	case IDInt64:
		return TypeInt64
	case IDUint64:
		return TypeUint64
	case IDFloat32:
		return TypeFloat32
	case IDFloat64:
		return TypeFloat64
	case IDBytes:
		return TypeBytes
	case IDString:
		return TypeString
	case IDIP:
		return TypeIP
	case IDNet:
		return TypeNet
	case IDTime:
		return TypeTime
	case IDDuration:
		return TypeDuration
	case IDType:
		return TypeType
	case IDNull:
		return TypeNull
	}
	return nil
}

// Utilities shared by complex types (ie, set and array)

// InnerType returns the element type for the underlying set or array type or
// nil if the underlying type is not a set or array.
func InnerType(typ Type) Type {
	switch typ := TypeUnder(typ).(type) {
	case *TypeSet:
		return typ.Type
	case *TypeArray:
		return typ.Type
	default:
		return nil
	}
}

// ContainedType returns the inner type for set and array types in the first
// return value and the columns of its of type for record types in the second
// return value.  ContainedType returns nil for both return values if the
// type is not a set, array, or record.
func ContainedType(typ Type) (Type, []Column) {
	switch typ := typ.(type) {
	case *TypeSet:
		return typ.Type, nil
	case *TypeArray:
		return typ.Type, nil
	case *TypeRecord:
		return nil, typ.Columns
		// XXX enum?
	default:
		return nil, nil
	}
}

func IsUnionType(typ Type) bool {
	_, ok := typ.(*TypeUnion)
	return ok
}

func IsRecordType(typ Type) bool {
	_, ok := TypeUnder(typ).(*TypeRecord)
	return ok
}

func TypeRecordOf(typ Type) *TypeRecord {
	t, _ := TypeUnder(typ).(*TypeRecord)
	return t
}

func IsContainerType(typ Type) bool {
	switch typ := typ.(type) {
	case *TypeAlias:
		return IsContainerType(typ.Type)
	case *TypeSet, *TypeArray, *TypeRecord, *TypeUnion, *TypeMap:
		return true
	default:
		return false
	}
}

func IsPrimitiveType(typ Type) bool {
	return !IsContainerType(typ)
}

func TypeID(typ Type) int {
	if alias, ok := typ.(*TypeAlias); ok {
		return alias.id
	}
	return typ.ID()
}

package zed21

import "errors"

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
	TypeBool    = &TypeOfBool{}
	TypeBytes   = &TypeOfBytes{}
	TypeString  = &TypeOfString{}
	TypeBstring = &TypeOfBstring{}
	TypeIP      = &TypeOfIP{}
	TypeNet     = &TypeOfNet{}
	TypeType    = &TypeOfType{}
	TypeError   = &TypeOfError{}
	TypeNull    = &TypeOfNull{}
)

const (
	IDUint8    = 0
	IDUint16   = 1
	IDUint32   = 2
	IDUint64   = 3
	IDInt8     = 4
	IDInt16    = 5
	IDInt32    = 6
	IDInt64    = 7
	IDDuration = 8
	IDTime     = 9
	IDFloat16  = 10
	IDFloat32  = 11
	IDFloat64  = 12
	IDDecimal  = 13
	IDBool     = 14
	IDBytes    = 15
	IDString   = 16
	IDBstring  = 17
	IDIP       = 18
	IDNet      = 19
	IDType     = 20
	IDError    = 21
	IDNull     = 22

	IDTypeDef = 23 // 0x17

	IDTypeName   = 24 // 0x18
	IDTypeRecord = 25 // 0x19
	IDTypeArray  = 26 // 0x20
	IDTypeSet    = 27 // 0x21
	IDTypeUnion  = 28 // 0x22
	IDTypeEnum   = 29 // 0x23
	IDTypeMap    = 30 // 0x24
)

const (
	CtrlValueEscape   = 0xf5
	TypeDefRecord     = 0xf6
	TypeDefArray      = 0xf7
	TypeDefSet        = 0xf8
	TypeDefUnion      = 0xf9
	TypeDefEnum       = 0xfa
	TypeDefMap        = 0xfb
	TypeDefAlias      = 0xfc
	CtrlCompressed    = 0xfd
	CtrlAppMessage    = 0xfe
	CtrlEOS           = 0xff
	AppEncodingZNG    = 0
	AppEncodingJSON   = 1
	AppEncodingZSON   = 2
	AppEncodingString = 3
	AppEncodingBinary = 4
)

type CompressionFormat int

const CompressionFormatLZ4 CompressionFormat = 0x00

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
	case "bstring":
		return TypeBstring
	case "ip":
		return TypeIP
	case "net":
		return TypeNet
	case "type":
		return TypeType
	case "error":
		return TypeError
	case "null":
		return TypeNull
	}
	return nil
}

func LookupPrimitiveByID(id int) Type {
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
	case IDBstring:
		return TypeBstring
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
	case IDError:
		return TypeError
	case IDNull:
		return TypeNull
	}
	return nil
}

// Utilities shared by complex types (ie, set and array)

// InnerType returns the element type for the underlying set or array type or
// nil if the underlying type is not a set or array.
func InnerType(typ Type) Type {
	switch typ := AliasOf(typ).(type) {
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
	_, ok := AliasOf(typ).(*TypeRecord)
	return ok
}

func TypeRecordOf(typ Type) *TypeRecord {
	t, _ := AliasOf(typ).(*TypeRecord)
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

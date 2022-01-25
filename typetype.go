package zed

import (
	"github.com/brimdata/zed/zcode"
)

type TypeOfType struct{}

func (t *TypeOfType) ID() int {
	return IDType
}

func (t *TypeOfType) Kind() Kind {
	return PrimitiveKind
}

func NewTypeValue(t Type) *Value {
	return &Value{TypeType, EncodeTypeValue(t)}
}

func EncodeTypeValue(t Type) zcode.Bytes {
	var typedefs map[string]Type
	return appendTypeValue(nil, t, &typedefs)
}

func appendTypeValue(b zcode.Bytes, t Type, typedefs *map[string]Type) zcode.Bytes {
	switch t := t.(type) {
	case *TypeAlias:
		if *typedefs == nil {
			*typedefs = make(map[string]Type)
		}
		id := byte(TypeValueNameDef)
		if previous := (*typedefs)[t.Name]; previous == t.Type {
			id = TypeValueNameRef
		} else {
			(*typedefs)[t.Name] = t.Type
		}
		b = append(b, id)
		b = zcode.AppendUvarint(b, uint64(len(t.Name)))
		b = append(b, zcode.Bytes(t.Name)...)
		if id == TypeValueNameRef {
			return b
		}
		return appendTypeValue(b, t.Type, typedefs)
	case *TypeRecord:
		b = append(b, TypeValueRecord)
		b = zcode.AppendUvarint(b, uint64(len(t.Columns)))
		for _, col := range t.Columns {
			b = zcode.AppendUvarint(b, uint64(len(col.Name)))
			b = append(b, col.Name...)
			b = appendTypeValue(b, col.Type, typedefs)
		}
		return b
	case *TypeUnion:
		b = append(b, TypeValueUnion)
		b = zcode.AppendUvarint(b, uint64(len(t.Types)))
		for _, t := range t.Types {
			b = appendTypeValue(b, t, typedefs)
		}
		return b
	case *TypeSet:
		b = append(b, TypeValueSet)
		return appendTypeValue(b, t.Type, typedefs)
	case *TypeArray:
		b = append(b, TypeValueArray)
		return appendTypeValue(b, t.Type, typedefs)
	case *TypeEnum:
		b = append(b, TypeValueEnum)
		b = zcode.AppendUvarint(b, uint64(len(t.Symbols)))
		for _, s := range t.Symbols {
			b = zcode.AppendUvarint(b, uint64(len(s)))
			b = append(b, s...)
		}
		return b
	case *TypeMap:
		b = append(b, TypeValueMap)
		b = appendTypeValue(b, t.KeyType, typedefs)
		return appendTypeValue(b, t.ValType, typedefs)
	case *TypeError:
		b = append(b, TypeValueError)
		return appendTypeValue(b, t.Type, typedefs)
	default:
		// Primitive type
		return append(b, byte(t.ID()))
	}
}

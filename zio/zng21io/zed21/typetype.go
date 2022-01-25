package zed21

import (
	"encoding/binary"

	"github.com/brimdata/zed/zcode"
)

type TypeOfType struct{}

func (t *TypeOfType) ID() int {
	return IDType
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
		id := byte(IDTypeDef)
		if previous := (*typedefs)[t.Name]; previous == t.Type {
			id = IDTypeName
		} else {
			(*typedefs)[t.Name] = t.Type
		}
		b = append(b, id)
		b = zcode.AppendUvarint(b, uint64(len(t.Name)))
		b = append(b, zcode.Bytes(t.Name)...)
		if id == IDTypeName {
			return b
		}
		return appendTypeValue(b, t.Type, typedefs)
	case *TypeRecord:
		b = append(b, IDTypeRecord)
		b = zcode.AppendUvarint(b, uint64(len(t.Columns)))
		for _, col := range t.Columns {
			b = zcode.AppendUvarint(b, uint64(len(col.Name)))
			b = append(b, col.Name...)
			b = appendTypeValue(b, col.Type, typedefs)
		}
		return b
	case *TypeUnion:
		b = append(b, IDTypeUnion)
		b = zcode.AppendUvarint(b, uint64(len(t.Types)))
		for _, t := range t.Types {
			b = appendTypeValue(b, t, typedefs)
		}
		return b
	case *TypeSet:
		b = append(b, IDTypeSet)
		return appendTypeValue(b, t.Type, typedefs)
	case *TypeArray:
		b = append(b, IDTypeArray)
		return appendTypeValue(b, t.Type, typedefs)
	case *TypeEnum:
		b = append(b, IDTypeEnum)
		b = zcode.AppendUvarint(b, uint64(len(t.Symbols)))
		for _, s := range t.Symbols {
			b = zcode.AppendUvarint(b, uint64(len(s)))
			b = append(b, s...)
		}
		return b
	case *TypeMap:
		b = append(b, IDTypeMap)
		b = appendTypeValue(b, t.KeyType, typedefs)
		return appendTypeValue(b, t.ValType, typedefs)
	default:
		// Primitive type
		return append(b, byte(t.ID()))
	}
}

func decodeName(tv zcode.Bytes) (string, zcode.Bytes) {
	namelen, tv := decodeInt(tv)
	if tv == nil || int(namelen) > len(tv) {
		return "", nil
	}
	return string(tv[:namelen]), tv[namelen:]
}

func decodeInt(tv zcode.Bytes) (int, zcode.Bytes) {
	if len(tv) < 0 {
		return 0, nil
	}
	namelen, n := binary.Uvarint(tv)
	if n <= 0 {
		return 0, nil
	}
	return int(namelen), tv[n:]
}

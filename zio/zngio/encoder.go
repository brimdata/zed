package zngio

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Encoder struct {
	zctx    *zed.Context
	encoded map[zed.Type]zed.Type
}

func NewEncoder() *Encoder {
	return &Encoder{
		zctx:    zed.NewContext(),
		encoded: make(map[zed.Type]zed.Type),
	}
}

func (e *Encoder) Reset() {
	e.encoded = make(map[zed.Type]zed.Type)
	e.zctx.Reset()
}

func (e *Encoder) Lookup(external zed.Type) zed.Type {
	return e.encoded[external]
}

// Encode takes a type from outside this context and constructs a type from
// inside this context and emits ZNG typedefs for any type needed to construct
// the new type into the buffer provided.
func (e *Encoder) Encode(dst []byte, external zed.Type) ([]byte, zed.Type, error) {
	if typ, ok := e.encoded[external]; ok {
		return dst, typ, nil
	}
	dst, internal, err := e.encode(dst, external)
	if err != nil {
		return nil, nil, err
	}
	e.encoded[external] = internal
	return dst, internal, err
}

func (e *Encoder) encode(dst []byte, ext zed.Type) ([]byte, zed.Type, error) {
	switch ext := ext.(type) {
	case *zed.TypeRecord:
		return e.encodeTypeRecord(dst, ext)
	case *zed.TypeSet:
		return e.encodeTypeSet(dst, ext)
	case *zed.TypeArray:
		return e.encodeTypeArray(dst, ext)
	case *zed.TypeUnion:
		return e.encodeTypeUnion(dst, ext)
	case *zed.TypeMap:
		return e.encodeTypeMap(dst, ext)
	case *zed.TypeEnum:
		return e.encodeTypeEnum(dst, ext)
	case *zed.TypeAlias:
		return e.encodeTypeAlias(dst, ext)
	default:
		return dst, ext, nil
	}
}

func (e *Encoder) encodeTypeRecord(dst []byte, ext *zed.TypeRecord) ([]byte, zed.Type, error) {
	var columns []zed.Column
	for _, col := range ext.Columns {
		var child zed.Type
		var err error
		dst, child, err = e.Encode(dst, col.Type)
		if err != nil {
			return nil, nil, err
		}
		columns = append(columns, zed.NewColumn(col.Name, child))
	}
	typ, err := e.zctx.LookupTypeRecord(columns)
	if err != nil {
		return nil, nil, err
	}
	dst = append(dst, zed.TypeDefRecord)
	dst = zcode.AppendUvarint(dst, uint64(len(columns)))
	for _, col := range columns {
		name := []byte(col.Name)
		dst = zcode.AppendUvarint(dst, uint64(len(name)))
		dst = append(dst, name...)
		dst = zcode.AppendUvarint(dst, uint64(zed.TypeID(col.Type)))
	}
	return dst, typ, nil
}

func (e *Encoder) encodeTypeUnion(dst []byte, ext *zed.TypeUnion) ([]byte, zed.Type, error) {
	var types []zed.Type
	for _, t := range ext.Types {
		var err error
		dst, t, err = e.Encode(dst, t)
		if err != nil {
			return nil, nil, err
		}
		types = append(types, t)
	}
	typ := e.zctx.LookupTypeUnion(types)
	dst = append(dst, zed.TypeDefUnion)
	dst = zcode.AppendUvarint(dst, uint64(len(types)))
	for _, t := range types {
		dst = zcode.AppendUvarint(dst, uint64(zed.TypeID(t)))
	}
	return dst, typ, nil
}

func (e *Encoder) encodeTypeSet(dst []byte, ext *zed.TypeSet) ([]byte, zed.Type, error) {
	var inner zed.Type
	var err error
	dst, inner, err = e.Encode(dst, ext.Type)
	if err != nil {
		return nil, nil, err
	}
	typ := e.zctx.LookupTypeSet(inner)
	dst = append(dst, zed.TypeDefSet)
	return zcode.AppendUvarint(dst, uint64(zed.TypeID(inner))), typ, nil
}

func (e *Encoder) encodeTypeArray(dst []byte, ext *zed.TypeArray) ([]byte, zed.Type, error) {
	var inner zed.Type
	var err error
	dst, inner, err = e.Encode(dst, ext.Type)
	if err != nil {
		return nil, nil, err
	}
	typ := e.zctx.LookupTypeArray(inner)
	dst = append(dst, zed.TypeDefArray)
	return zcode.AppendUvarint(dst, uint64(zed.TypeID(inner))), typ, nil
}

func (e *Encoder) encodeTypeEnum(dst []byte, ext *zed.TypeEnum) ([]byte, zed.Type, error) {
	symbols := ext.Symbols
	typ := e.zctx.LookupTypeEnum(symbols)
	dst = append(dst, zed.TypeDefEnum)
	dst = zcode.AppendUvarint(dst, uint64(len(symbols)))
	for _, s := range symbols {
		dst = zcode.AppendUvarint(dst, uint64(len(s)))
		dst = append(dst, s...)
	}
	return dst, typ, nil
}

func (e *Encoder) encodeTypeMap(dst []byte, ext *zed.TypeMap) ([]byte, zed.Type, error) {
	var keyType zed.Type
	var err error
	dst, keyType, err = e.Encode(dst, ext.KeyType)
	if err != nil {
		return nil, nil, err
	}
	var valType zed.Type
	dst, valType, err = e.Encode(dst, ext.ValType)
	if err != nil {
		return nil, nil, err
	}
	typ := e.zctx.LookupTypeMap(keyType, valType)
	dst = append(dst, zed.TypeDefMap)
	dst = zcode.AppendUvarint(dst, uint64(zed.TypeID(keyType)))
	return zcode.AppendUvarint(dst, uint64(zed.TypeID(valType))), typ, nil
}

func (e *Encoder) encodeTypeAlias(dst []byte, ext *zed.TypeAlias) ([]byte, zed.Type, error) {
	var inner zed.Type
	var err error
	dst, inner, err = e.Encode(dst, ext.Type)
	if err != nil {
		return nil, nil, err
	}
	typ, err := e.zctx.LookupTypeAlias(ext.Name, inner)
	if err != nil {
		return nil, nil, err
	}
	dst = append(dst, zed.TypeDefAlias)
	dst = zcode.AppendUvarint(dst, uint64(len(typ.Name)))
	dst = append(dst, typ.Name...)
	return zcode.AppendUvarint(dst, uint64(zed.TypeID(typ.Type))), typ, nil
}

package zngio

import (
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

type Encoder struct {
	zctx    *zson.Context
	encoded map[zng.Type]zng.Type
}

func NewEncoder() *Encoder {
	return &Encoder{
		zctx:    zson.NewContext(),
		encoded: make(map[zng.Type]zng.Type),
	}
}

func (e *Encoder) Reset() {
	e.encoded = make(map[zng.Type]zng.Type)
	e.zctx.Reset()
}

func (e *Encoder) Lookup(external zng.Type) zng.Type {
	return e.encoded[external]
}

// Encode takes a type from outside this context and constructs a type from
// inside this context and emits ZNG typedefs for any type needed to construct
// the new type into the buffer provided.
func (e *Encoder) Encode(dst []byte, external zng.Type) ([]byte, zng.Type, error) {
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

func (e *Encoder) encode(dst []byte, ext zng.Type) ([]byte, zng.Type, error) {
	switch ext := ext.(type) {
	case *zng.TypeRecord:
		return e.encodeTypeRecord(dst, ext)
	case *zng.TypeSet:
		return e.encodeTypeSet(dst, ext)
	case *zng.TypeArray:
		return e.encodeTypeArray(dst, ext)
	case *zng.TypeUnion:
		return e.encodeTypeUnion(dst, ext)
	case *zng.TypeMap:
		return e.encodeTypeMap(dst, ext)
	case *zng.TypeEnum:
		return e.encodeTypeEnum(dst, ext)
	case *zng.TypeAlias:
		return e.encodeTypeAlias(dst, ext)
	default:
		return dst, ext, nil
	}
}

func (e *Encoder) encodeTypeRecord(dst []byte, ext *zng.TypeRecord) ([]byte, zng.Type, error) {
	var columns []zng.Column
	for _, col := range ext.Columns {
		var child zng.Type
		var err error
		dst, child, err = e.Encode(dst, col.Type)
		if err != nil {
			return nil, nil, err
		}
		columns = append(columns, zng.NewColumn(col.Name, child))
	}
	typ, err := e.zctx.LookupTypeRecord(columns)
	if err != nil {
		return nil, nil, err
	}
	dst = append(dst, zng.TypeDefRecord)
	dst = zcode.AppendUvarint(dst, uint64(len(columns)))
	for _, col := range columns {
		name := []byte(col.Name)
		dst = zcode.AppendUvarint(dst, uint64(len(name)))
		dst = append(dst, name...)
		dst = zcode.AppendUvarint(dst, uint64(zng.TypeID(col.Type)))
	}
	return dst, typ, nil
}

func (e *Encoder) encodeTypeUnion(dst []byte, ext *zng.TypeUnion) ([]byte, zng.Type, error) {
	var types []zng.Type
	for _, t := range ext.Types {
		var err error
		dst, t, err = e.Encode(dst, t)
		if err != nil {
			return nil, nil, err
		}
		types = append(types, t)
	}
	typ := e.zctx.LookupTypeUnion(types)
	dst = append(dst, zng.TypeDefUnion)
	dst = zcode.AppendUvarint(dst, uint64(len(types)))
	for _, t := range types {
		dst = zcode.AppendUvarint(dst, uint64(zng.TypeID(t)))
	}
	return dst, typ, nil
}

func (e *Encoder) encodeTypeSet(dst []byte, ext *zng.TypeSet) ([]byte, zng.Type, error) {
	var inner zng.Type
	var err error
	dst, inner, err = e.Encode(dst, ext.Type)
	if err != nil {
		return nil, nil, err
	}
	typ := e.zctx.LookupTypeSet(inner)
	dst = append(dst, zng.TypeDefSet)
	return zcode.AppendUvarint(dst, uint64(zng.TypeID(inner))), typ, nil
}

func (e *Encoder) encodeTypeArray(dst []byte, ext *zng.TypeArray) ([]byte, zng.Type, error) {
	var inner zng.Type
	var err error
	dst, inner, err = e.Encode(dst, ext.Type)
	if err != nil {
		return nil, nil, err
	}
	typ := e.zctx.LookupTypeArray(inner)
	dst = append(dst, zng.TypeDefArray)
	return zcode.AppendUvarint(dst, uint64(zng.TypeID(inner))), typ, nil
}

func (e *Encoder) encodeTypeEnum(dst []byte, ext *zng.TypeEnum) ([]byte, zng.Type, error) {
	symbols := ext.Symbols
	typ := e.zctx.LookupTypeEnum(symbols)
	dst = append(dst, zng.TypeDefEnum)
	dst = zcode.AppendUvarint(dst, uint64(len(symbols)))
	for _, s := range symbols {
		dst = zcode.AppendUvarint(dst, uint64(len(s)))
		dst = append(dst, s...)
	}
	return dst, typ, nil
}

func (e *Encoder) encodeTypeMap(dst []byte, ext *zng.TypeMap) ([]byte, zng.Type, error) {
	var keyType zng.Type
	var err error
	dst, keyType, err = e.Encode(dst, ext.KeyType)
	if err != nil {
		return nil, nil, err
	}
	var valType zng.Type
	dst, valType, err = e.Encode(dst, ext.ValType)
	if err != nil {
		return nil, nil, err
	}
	typ := e.zctx.LookupTypeMap(keyType, valType)
	dst = append(dst, zng.TypeDefMap)
	dst = zcode.AppendUvarint(dst, uint64(zng.TypeID(keyType)))
	return zcode.AppendUvarint(dst, uint64(zng.TypeID(valType))), typ, nil
}

func (e *Encoder) encodeTypeAlias(dst []byte, ext *zng.TypeAlias) ([]byte, zng.Type, error) {
	var inner zng.Type
	var err error
	dst, inner, err = e.Encode(dst, ext.Type)
	if err != nil {
		return nil, nil, err
	}
	typ, err := e.zctx.LookupTypeAlias(ext.Name, inner)
	if err != nil {
		return nil, nil, err
	}
	dst = append(dst, zng.TypeDefAlias)
	dst = zcode.AppendUvarint(dst, uint64(len(typ.Name)))
	dst = append(dst, typ.Name...)
	return zcode.AppendUvarint(dst, uint64(zng.TypeID(typ.Type))), typ, nil
}

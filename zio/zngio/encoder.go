package zngio

import (
	"fmt"

	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

type Encoder struct {
	table   []zng.Type
	zctx    *zson.Context
	encoded map[zng.Type]struct{}
}

func NewEncoder() *Encoder {
	return &Encoder{
		zctx:    zson.NewContext(),
		encoded: make(map[zng.Type]struct{}),
	}
}

func (e *Encoder) Reset() {
	e.table = e.table[:0]
	e.encoded = make(map[zng.Type]struct{})
	e.zctx.Reset()
}

func (e *Encoder) Lookup(external zng.Type) zng.Type {
	id := external.ID()
	if id >= 0 && id < len(e.table) {
		return e.table[id]
	}
	return nil
}

func (e *Encoder) enter(id int, typ zng.Type) {
	if id >= len(e.table) {
		new := make([]zng.Type, id+1)
		copy(new, e.table)
		e.table = new
	}
	e.table[id] = typ
}

func (e *Encoder) isEncoded(t zng.Type) bool {
	_, ok := e.encoded[t]
	if !ok {
		e.encoded[t] = struct{}{}
	}
	return ok
}

// Encode takes a type from outside this context and constructs a type from
// inside this context and emits ZNG typedefs for any type needed to construct
// the new type into the buffer provided.
func (e *Encoder) Encode(dst []byte, external zng.Type) ([]byte, zng.Type, error) {
	dst, typ, err := e.encodeType(dst, external)
	if err != nil {
		return nil, nil, err
	}
	e.enter(external.ID(), typ)
	return dst, typ, err
}

func (e *Encoder) encodeType(dst []byte, ext zng.Type) ([]byte, zng.Type, error) {
	id := ext.ID()
	if _, alias := ext.(*zng.TypeAlias); id < zng.IDTypeDef && !alias {
		return dst, ext, nil
	}
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
		//XXX
		panic(fmt.Sprintf("zng cannot encode type: %s", ext))
	}
}

func (e *Encoder) encodeTypeRecord(dst []byte, ext *zng.TypeRecord) ([]byte, zng.Type, error) {
	var columns []zng.Column
	for _, col := range ext.Columns {
		var child zng.Type
		var err error
		dst, child, err = e.encodeType(dst, col.Type)
		if err != nil {
			return nil, nil, err
		}
		columns = append(columns, zng.NewColumn(col.Name, child))
	}
	typ, err := e.zctx.LookupTypeRecord(columns)
	if err != nil {
		return nil, nil, err
	}
	if e.isEncoded(typ) {
		return dst, typ, nil
	}
	return serializeTypeRecord(dst, columns), typ, nil
}

func serializeTypeRecord(dst []byte, columns []zng.Column) []byte {
	dst = append(dst, zng.TypeDefRecord)
	dst = zcode.AppendUvarint(dst, uint64(len(columns)))
	for _, col := range columns {
		name := []byte(col.Name)
		dst = zcode.AppendUvarint(dst, uint64(len(name)))
		dst = append(dst, name...)
		dst = zcode.AppendUvarint(dst, uint64(zng.TypeID(col.Type)))
	}
	return dst
}

func (e *Encoder) encodeTypeUnion(dst []byte, ext *zng.TypeUnion) ([]byte, zng.Type, error) {
	var types []zng.Type
	for _, t := range ext.Types {
		var err error
		dst, t, err = e.encodeType(dst, t)
		if err != nil {
			return nil, nil, err
		}
		types = append(types, t)
	}
	typ := e.zctx.LookupTypeUnion(types)
	if e.isEncoded(typ) {
		return dst, typ, nil
	}
	return serializeTypeUnion(dst, types), typ, nil
}

func serializeTypeUnion(dst []byte, types []zng.Type) []byte {
	dst = append(dst, zng.TypeDefUnion)
	dst = zcode.AppendUvarint(dst, uint64(len(types)))
	for _, t := range types {
		dst = zcode.AppendUvarint(dst, uint64(zng.TypeID(t)))
	}
	return dst
}

func (e *Encoder) encodeTypeSet(dst []byte, ext *zng.TypeSet) ([]byte, zng.Type, error) {
	var inner zng.Type
	var err error
	dst, inner, err = e.encodeType(dst, ext.Type)
	if err != nil {
		return nil, nil, err
	}
	typ := e.zctx.LookupTypeSet(inner)
	if e.isEncoded(typ) {
		return dst, typ, nil
	}
	return serializeTypeSet(dst, typ.Type), typ, nil
}

func serializeTypeSet(dst []byte, inner zng.Type) []byte {
	dst = append(dst, zng.TypeDefSet)
	return zcode.AppendUvarint(dst, uint64(zng.TypeID(inner)))
}

func (e *Encoder) encodeTypeArray(dst []byte, ext *zng.TypeArray) ([]byte, zng.Type, error) {
	var inner zng.Type
	var err error
	dst, inner, err = e.encodeType(dst, ext.Type)
	if err != nil {
		return nil, nil, err
	}
	typ := e.zctx.LookupTypeArray(inner)
	if e.isEncoded(typ) {
		return dst, typ, nil
	}
	return serializeTypeArray(dst, inner), typ, nil
}

func serializeTypeArray(dst []byte, inner zng.Type) []byte {
	dst = append(dst, zng.TypeDefArray)
	return zcode.AppendUvarint(dst, uint64(zng.TypeID(inner)))
}

func (e *Encoder) encodeTypeEnum(dst []byte, ext *zng.TypeEnum) ([]byte, zng.Type, error) {
	var elemType zng.Type
	var err error
	dst, elemType, err = e.encodeType(dst, ext.Type)
	if err != nil {
		return nil, nil, err
	}
	typ := e.zctx.LookupTypeEnum(elemType, ext.Elements)
	if e.isEncoded(typ) {
		return dst, typ, nil
	}
	return serializeTypeEnum(dst, elemType, typ.Elements), typ, nil
}

func serializeTypeEnum(dst []byte, typ zng.Type, elems []zng.Element) []byte {
	dst = append(dst, zng.TypeDefEnum)
	dst = zcode.AppendUvarint(dst, uint64(zng.TypeID(typ)))
	dst = zcode.AppendUvarint(dst, uint64(len(elems)))
	container := zng.IsContainerType(typ)
	for _, elem := range elems {
		name := []byte(elem.Name)
		dst = zcode.AppendUvarint(dst, uint64(len(name)))
		dst = append(dst, name...)
		dst = zcode.AppendAs(dst, container, elem.Value)
	}
	return dst
}

func (e *Encoder) encodeTypeMap(dst []byte, ext *zng.TypeMap) ([]byte, zng.Type, error) {
	var keyType zng.Type
	var err error
	dst, keyType, err = e.encodeType(dst, ext.KeyType)
	if err != nil {
		return nil, nil, err
	}
	var valType zng.Type
	dst, valType, err = e.encodeType(dst, ext.ValType)
	if err != nil {
		return nil, nil, err
	}
	typ := e.zctx.LookupTypeMap(keyType, valType)
	if e.isEncoded(typ) {
		return dst, typ, nil
	}
	return serializeTypeMap(dst, keyType, valType), typ, nil
}

func serializeTypeMap(dst []byte, keyType, valType zng.Type) []byte {
	dst = append(dst, zng.TypeDefMap)
	dst = zcode.AppendUvarint(dst, uint64(zng.TypeID(keyType)))
	return zcode.AppendUvarint(dst, uint64(zng.TypeID(valType)))
}

func (e *Encoder) encodeTypeAlias(dst []byte, ext *zng.TypeAlias) ([]byte, zng.Type, error) {
	var inner zng.Type
	var err error
	dst, inner, err = e.encodeType(dst, ext.Type)
	if err != nil {
		return nil, nil, err
	}
	typ, err := e.zctx.LookupTypeAlias(ext.Name, inner)
	if err != nil {
		return nil, nil, err
	}
	if e.isEncoded(typ) {
		return dst, typ, nil
	}
	return serializeTypeAlias(dst, typ), typ, nil
}

func serializeTypeAlias(dst []byte, alias *zng.TypeAlias) []byte {
	dst = append(dst, zng.TypeDefAlias)
	dst = zcode.AppendUvarint(dst, uint64(len(alias.Name)))
	dst = append(dst, alias.Name...)
	return zcode.AppendUvarint(dst, uint64(zng.TypeID(alias.Type)))
}

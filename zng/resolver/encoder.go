package resolver

import (
	"fmt"

	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zng"
)

type Encoder struct {
	table   []zng.Type
	zctx    *Context
	encoded map[int]struct{}
}

func NewEncoder() *Encoder {
	return &Encoder{
		zctx:    NewContext(),
		encoded: make(map[int]struct{}),
	}
}

func (e *Encoder) Reset() {
	e.table = e.table[:0]
	e.encoded = make(map[int]struct{})
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

func (e *Encoder) isEncoded(id int) bool {
	if _, ok := e.encoded[id]; ok {
		return true
	}
	e.encoded[id] = struct{}{}
	return false
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
	if _, alias := ext.(*zng.TypeAlias); id < zng.IdTypeDef && !alias {
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
	if e.isEncoded(typ.ID()) {
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
		if typ, ok := col.Type.(*zng.TypeAlias); ok {
			dst = zcode.AppendUvarint(dst, uint64(typ.AliasID()))
		} else {
			dst = zcode.AppendUvarint(dst, uint64(col.Type.ID()))
		}
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
	if e.isEncoded(typ.ID()) {
		return dst, typ, nil
	}
	return serializeTypeUnion(dst, types), typ, nil
}

func serializeTypeUnion(dst []byte, types []zng.Type) []byte {
	dst = append(dst, zng.TypeDefUnion)
	dst = zcode.AppendUvarint(dst, uint64(len(types)))
	for _, t := range types {
		dst = zcode.AppendUvarint(dst, uint64(t.ID()))
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
	if e.isEncoded(typ.ID()) {
		return dst, typ, nil
	}
	return serializeTypeSet(dst, typ.Type), typ, nil
}

func serializeTypeSet(dst []byte, inner zng.Type) []byte {
	dst = append(dst, zng.TypeDefSet)
	return zcode.AppendUvarint(dst, uint64(inner.ID()))
}

func (e *Encoder) encodeTypeArray(dst []byte, ext *zng.TypeArray) ([]byte, zng.Type, error) {
	var inner zng.Type
	var err error
	dst, inner, err = e.encodeType(dst, ext.Type)
	if err != nil {
		return nil, nil, err
	}
	typ := e.zctx.LookupTypeArray(inner)
	if e.isEncoded(typ.ID()) {
		return dst, typ, nil
	}
	return serializeTypeArray(dst, inner), typ, nil
}

func serializeTypeArray(dst []byte, inner zng.Type) []byte {
	dst = append(dst, zng.TypeDefArray)
	return zcode.AppendUvarint(dst, uint64(inner.ID()))
}

func (e *Encoder) encodeTypeEnum(dst []byte, ext *zng.TypeEnum) ([]byte, zng.Type, error) {
	var elemType zng.Type
	var err error
	dst, elemType, err = e.encodeType(dst, ext.Type)
	if err != nil {
		return nil, nil, err
	}
	typ := e.zctx.LookupTypeEnum(elemType, ext.Elements)
	if e.isEncoded(typ.ID()) {
		return dst, typ, nil
	}
	return serializeTypeEnum(dst, elemType, typ.Elements), typ, nil
}

func serializeTypeEnum(dst []byte, typ zng.Type, elems []zng.Element) []byte {
	dst = append(dst, zng.TypeDefEnum)
	//XXX fix this alias business... zng.RealID()?
	if alias, ok := typ.(*zng.TypeAlias); ok {
		dst = zcode.AppendUvarint(dst, uint64(alias.AliasID()))
	} else {
		dst = zcode.AppendUvarint(dst, uint64(typ.ID()))
	}
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
	if e.isEncoded(typ.ID()) {
		return dst, typ, nil
	}
	return serializeTypeMap(dst, keyType, valType), typ, nil
}

func serializeTypeMap(dst []byte, keyType, valType zng.Type) []byte {
	dst = append(dst, zng.TypeDefMap)
	dst = zcode.AppendUvarint(dst, uint64(keyType.ID()))
	return zcode.AppendUvarint(dst, uint64(valType.ID()))
}

func serializeTypes(dst []byte, types []zng.Type) []byte {
	for _, typ := range types {
		switch typ := typ.(type) {
		case *zng.TypeRecord:
			dst = serializeTypeRecord(dst, typ.Columns)
		case *zng.TypeSet:
			dst = serializeTypeSet(dst, typ.Type)
		case *zng.TypeArray:
			dst = serializeTypeArray(dst, typ.Type)
		case *zng.TypeUnion:
			dst = serializeTypeUnion(dst, typ.Types)
		case *zng.TypeEnum:
			dst = serializeTypeEnum(dst, typ.Type, typ.Elements)
		case *zng.TypeMap:
			dst = serializeTypeMap(dst, typ.KeyType, typ.ValType)
		case *zng.TypeAlias:
			dst = serializeTypeAlias(dst, typ)
		default:
			panic(fmt.Sprintf("zng cannot serialize type: %s", typ))
		}
	}
	return dst
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
	if e.isEncoded(typ.AliasID()) {
		return dst, typ, nil
	}
	return serializeTypeAlias(dst, typ), typ, nil
}

func serializeTypeAlias(dst []byte, alias *zng.TypeAlias) []byte {
	dst = append(dst, zng.TypeDefAlias)
	dst = zcode.AppendUvarint(dst, uint64(len(alias.Name)))
	dst = append(dst, alias.Name...)
	// Need to check if target is another alias and call target.AliasID().
	// Otherwise calling target.ID() will recurse to the base target.
	if target, ok := alias.Type.(*zng.TypeAlias); ok {
		return zcode.AppendUvarint(dst, uint64(target.AliasID()))
	}
	return zcode.AppendUvarint(dst, uint64(alias.ID()))
}

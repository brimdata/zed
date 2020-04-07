package resolver

import (
	"fmt"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
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
	default:
		//XXX
		panic(fmt.Sprintf("bzng cannot encode type: %s", ext))
	case *zng.TypeRecord:
		return e.encodeTypeRecord(dst, ext)
	case *zng.TypeSet:
		return e.encodeTypeSet(dst, ext)
	case *zng.TypeArray:
		return e.encodeTypeArray(dst, ext)
	case *zng.TypeUnion:
		return e.encodeTypeUnion(dst, ext)
	case *zng.TypeAlias:
		return e.encodeTypeAlias(dst, ext)
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
	typ := e.zctx.LookupTypeRecord(columns)
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
	dst, inner, err = e.encodeType(dst, ext.InnerType)
	if err != nil {
		return nil, nil, err
	}
	typ := e.zctx.LookupTypeSet(inner)
	if e.isEncoded(typ.ID()) {
		return dst, typ, nil
	}
	return serializeTypeSet(dst, typ.InnerType), typ, nil
}

func serializeTypeSet(dst []byte, inner zng.Type) []byte {
	dst = append(dst, zng.TypeDefSet)
	dst = zcode.AppendUvarint(dst, 1)
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

func serializeTypes(dst []byte, types []zng.Type) []byte {
	for _, typ := range types {
		switch typ := typ.(type) {
		default:
			panic(fmt.Sprintf("bzng cannot serialize type: %s", typ))
		case *zng.TypeRecord:
			dst = serializeTypeRecord(dst, typ.Columns)
		case *zng.TypeSet:
			dst = serializeTypeSet(dst, typ.InnerType)
		case *zng.TypeArray:
			dst = serializeTypeArray(dst, typ.Type)
		case *zng.TypeUnion:
			dst = serializeTypeUnion(dst, typ.Types)
		case *zng.TypeAlias:
			dst = serializeTypeAlias(dst, typ)
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

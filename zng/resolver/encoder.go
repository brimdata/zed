package resolver

import (
	"fmt"

	"github.com/mccanne/zq/zcode"
	"github.com/mccanne/zq/zng"
)

type Encoder struct {
	table []zng.Type
	zctx  *Context
}

func NewEncoder() *Encoder {
	return &Encoder{zctx: NewContext()}
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

// Encode takes a type from outside this context and constructs a type from
// inside this context and emits ZNG typedefs for any type needed to construct
// the new type into the buffer provided.
func (e *Encoder) Encode(dst []byte, external zng.Type) ([]byte, zng.Type) {
	dst, typ := e.encodeType(dst, external)
	e.enter(external.ID(), typ)
	return dst, typ
}

func (e *Encoder) encodeType(dst []byte, ext zng.Type) ([]byte, zng.Type) {
	id := ext.ID()
	if id < zng.IdTypeDef {
		return dst, ext
	}
	switch ext := ext.(type) {
	default:
		//XXX
		panic(fmt.Sprintf("bzng cannot encode type: %s", ext))
	case *zng.TypeRecord:
		return e.encodeTypeRecord(dst, ext)
	case *zng.TypeSet:
		return e.encodeTypeSet(dst, ext)
	case *zng.TypeVector:
		return e.encodeTypeVector(dst, ext)
	}
}

func (e *Encoder) encodeTypeRecord(dst []byte, ext *zng.TypeRecord) ([]byte, zng.Type) {
	var columns []zng.Column
	for _, col := range ext.Columns {
		var child zng.Type
		dst, child = e.encodeType(dst, col.Type)
		columns = append(columns, zng.NewColumn(col.Name, child))
	}
	typ := e.zctx.LookupTypeRecord(columns)
	dst = append(dst, zng.TypeDefRecord)
	dst = zcode.AppendUvarint(dst, uint64(len(columns)))
	for _, col := range columns {
		name := []byte(col.Name)
		dst = zcode.AppendUvarint(dst, uint64(len(name)))
		dst = append(dst, name...)
		dst = zcode.AppendUvarint(dst, uint64(col.Type.ID()))
	}
	return dst, typ
}

func (e *Encoder) encodeTypeSet(dst []byte, ext *zng.TypeSet) ([]byte, zng.Type) {
	var inner zng.Type
	dst, inner = e.encodeType(dst, ext.InnerType)
	typ := e.zctx.LookupTypeSet(inner)
	dst = append(dst, zng.TypeDefSet)
	dst = zcode.AppendUvarint(dst, 1)
	return zcode.AppendUvarint(dst, uint64(typ.InnerType.ID())), typ
}

func (e *Encoder) encodeTypeVector(dst []byte, ext *zng.TypeVector) ([]byte, zng.Type) {
	var inner zng.Type
	dst, inner = e.encodeType(dst, ext.Type)
	typ := e.zctx.LookupTypeVector(inner)
	dst = append(dst, zng.TypeDefArray)
	return zcode.AppendUvarint(dst, uint64(typ.Type.ID())), typ
}

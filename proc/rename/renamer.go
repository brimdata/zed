// rename renames one or more fields in a record. A field can only be
// renamed within its own record. For example id.orig_h can be renamed
// to id.src, but it cannot be renamed to src. Renames are applied
// left to right; each rename observes the effect of all
package rename

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/field"
)

//XXX this is not a proc... should go in expr?

type Function struct {
	zctx *zed.Context
	// For the dst field name, we just store the leaf name since the
	// src path and the dst path are the same and only differ in the leaf name.
	srcs    field.List
	dsts    field.List
	typeMap map[int]*zed.TypeRecord
}

var _ expr.Applier = (*Function)(nil)

func NewFunction(zctx *zed.Context, srcs, dsts field.List) *Function {
	return &Function{zctx, srcs, dsts, make(map[int]*zed.TypeRecord)}
}

func (r *Function) dstType(typ *zed.TypeRecord, src, dst field.Path) *zed.TypeRecord {
	c, ok := typ.ColumnOfField(src[0])
	if !ok {
		return typ
	}
	var innerType zed.Type
	if len(src) > 1 {
		recType, ok := typ.Columns[c].Type.(*zed.TypeRecord)
		if !ok {
			return typ
		}
		innerType = r.dstType(recType, src[1:], dst[1:])
	} else {
		innerType = typ.Columns[c].Type
	}
	newcols := make([]zed.Column, len(typ.Columns))
	copy(newcols, typ.Columns)
	newcols[c] = zed.Column{Name: dst[0], Type: innerType}
	typ, err := r.zctx.LookupTypeRecord(newcols)
	if err != nil {
		panic(err)
	}
	return typ
}

func (r *Function) computeType(typ *zed.TypeRecord) *zed.TypeRecord {
	for k, dst := range r.dsts {
		typ = r.dstType(typ, r.srcs[k], dst)
	}
	return typ
}

func (r *Function) Eval(ctx expr.Context, this *zed.Value) *zed.Value {
	id := this.Type.ID()
	typ, ok := r.typeMap[id]
	if !ok {
		typ = r.computeType(zed.TypeRecordOf(this.Type))
		r.typeMap[id] = typ
	}
	out := this.Copy()
	return zed.NewValue(typ, out.Bytes)
}

func (_ *Function) String() string { return "rename" }

func (_ *Function) Warning() string { return "" }

// rename renames one or more fields in a record. A field can only be
// renamed within its own record. For example id.orig_h can be renamed
// to id.src, but it cannot be renamed to src. Renames are applied
// left to right; each rename observes the effect of all
package rename

import (
	"errors"
	"fmt"

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

func (r *Function) dstType(typ *zed.TypeRecord, src, dst field.Path) (*zed.TypeRecord, error) {
	c, ok := typ.ColumnOfField(src[0])
	if !ok {
		return typ, nil
	}
	var innerType zed.Type
	if len(src) > 1 {
		recType, ok := typ.Columns[c].Type.(*zed.TypeRecord)
		if !ok {
			return typ, nil
		}
		typ, err := r.dstType(recType, src[1:], dst[1:])
		if err != nil {
			return nil, err
		}
		innerType = typ
	} else {
		innerType = typ.Columns[c].Type
	}
	newcols := make([]zed.Column, len(typ.Columns))
	copy(newcols, typ.Columns)
	newcols[c] = zed.Column{Name: dst[0], Type: innerType}
	typ, err := r.zctx.LookupTypeRecord(newcols)
	if err != nil {
		if errors.Is(err, zed.ErrDuplicateFields) {
			return nil, fmt.Errorf("rename: %s", err)
		}
		panic(err)
	}
	return typ, nil
}

func (r *Function) computeType(typ *zed.TypeRecord) (*zed.TypeRecord, error) {
	for k, dst := range r.dsts {
		var err error
		typ, err = r.dstType(typ, r.srcs[k], dst)
		if err != nil {
			return nil, err
		}
	}
	return typ, nil
}

func (r *Function) Eval(ctx expr.Context, this *zed.Value) *zed.Value {
	id := this.Type.ID()
	typ, ok := r.typeMap[id]
	if !ok {
		var err error
		typ, err = r.computeType(zed.TypeRecordOf(this.Type))
		if err != nil {
			return ctx.CopyValue(zed.NewError(err))
		}
		r.typeMap[id] = typ
	}
	out := this.Copy()
	return ctx.NewValue(typ, out.Bytes)
}

func (_ *Function) String() string { return "rename" }

func (_ *Function) Warning() string { return "" }

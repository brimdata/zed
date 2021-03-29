// rename renames one or more fields in a record. A field can only be
// renamed within its own record. For example id.orig_h can be renamed
// to id.src, but it cannot be renamed to src. Renames are applied
// left to right; each rename observes the effect of all
package rename

import (
	"github.com/brimdata/zq/field"
	"github.com/brimdata/zq/proc"
	"github.com/brimdata/zq/zng"
	"github.com/brimdata/zq/zng/resolver"
)

var _ proc.Function = (*Function)(nil)

type Function struct {
	zctx *resolver.Context
	// For the dst field name, we just store the leaf name since the
	// src path and the dst path are the same and only differ in the leaf name.
	srcs    []field.Static
	dsts    []field.Static
	typeMap map[int]*zng.TypeRecord
}

func NewFunction(zctx *resolver.Context, srcs, dsts []field.Static) *Function {
	return &Function{zctx, srcs, dsts, make(map[int]*zng.TypeRecord)}
}

func (r *Function) dstType(typ *zng.TypeRecord, src, dst field.Static) (*zng.TypeRecord, error) {
	c, ok := typ.ColumnOfField(src[0])
	if !ok {
		return typ, nil
	}
	var innerType zng.Type
	if len(src) > 1 {
		recType, ok := typ.Columns[c].Type.(*zng.TypeRecord)
		if !ok {
			return typ, nil
		}
		var err error
		innerType, err = r.dstType(recType, src[1:], dst[1:])
		if err != nil {
			return nil, err
		}
	} else {
		innerType = typ.Columns[c].Type
	}
	newcols := make([]zng.Column, len(typ.Columns))
	copy(newcols, typ.Columns)
	newcols[c] = zng.Column{Name: dst[0], Type: innerType}
	return r.zctx.LookupTypeRecord(newcols)
}

func (r *Function) computeType(typ *zng.TypeRecord) (*zng.TypeRecord, error) {
	var err error
	for k, dst := range r.dsts {
		typ, err = r.dstType(typ, r.srcs[k], dst)
		if err != nil {
			return nil, err
		}
	}
	return typ, nil
}

func (r *Function) Apply(in *zng.Record) (*zng.Record, error) {
	id := in.Type.ID()
	if _, ok := r.typeMap[id]; !ok {
		typ, err := r.computeType(zng.TypeRecordOf(in.Type))
		if err != nil {
			return nil, err
		}
		r.typeMap[id] = typ
	}
	out := in.Keep()
	return zng.NewRecord(r.typeMap[id], out.Bytes), nil
}

func (_ *Function) String() string { return "rename" }

func (_ *Function) Warning() string { return "" }

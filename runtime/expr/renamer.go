package expr

import (
	"errors"
	"fmt"
	"slices"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
)

// Renamer renames one or more fields in a record. A field can only be
// renamed within its own record. For example id.orig_h can be renamed
// to id.src, but it cannot be renamed to src. Renames are applied
// left to right; each rename observes the effect of all.
type Renamer struct {
	zctx *zed.Context
	// For the dst field name, we just store the leaf name since the
	// src path and the dst path are the same and only differ in the leaf name.
	srcs    []*Lval
	dsts    []*Lval
	typeMap map[int]map[string]*zed.TypeRecord
	// fieldsStr is used to reduce allocations when computing the fields id.
	fieldsStr []byte
}

func NewRenamer(zctx *zed.Context, srcs, dsts []*Lval) *Renamer {
	return &Renamer{zctx, srcs, dsts, make(map[int]map[string]*zed.TypeRecord), nil}
}

func (r *Renamer) Eval(ectx Context, this *zed.Value) *zed.Value {
	if !zed.IsRecordType(this.Type) {
		return this
	}
	srcs, dsts, err := r.evalFields(ectx, this)
	if err != nil {
		return ectx.CopyValue(*r.zctx.WrapError(fmt.Sprintf("rename: %s", err), this))
	}
	id := this.Type.ID()
	m, ok := r.typeMap[id]
	if !ok {
		m = make(map[string]*zed.TypeRecord)
		r.typeMap[id] = m
	}
	r.fieldsStr = dsts.AppendTo(srcs.AppendTo(r.fieldsStr[:0]))
	typ, ok := m[string(r.fieldsStr)]
	if !ok {
		var err error
		typ, err = r.computeType(zed.TypeRecordOf(this.Type), srcs, dsts)
		if err != nil {
			return ectx.CopyValue(*r.zctx.WrapError(fmt.Sprintf("rename: %s", err), this))
		}
		m[string(r.fieldsStr)] = typ
	}
	return ectx.NewValue(typ, this.Bytes())
}

func CheckRenameField(src, dst field.Path) error {
	if len(src) != len(dst) {
		return fmt.Errorf("left-hand side and right-hand side must have the same depth (%s vs %s)", src, dst)
	}
	for i := 0; i <= len(src)-2; i++ {
		if src[i] != dst[i] {
			return fmt.Errorf("cannot rename %s to %s (differ in %s vs %s)", src, dst, src[i], dst[i])
		}
	}
	return nil
}

func (r *Renamer) evalFields(ectx Context, this *zed.Value) (field.List, field.List, error) {
	var srcs, dsts field.List
	for i := range r.srcs {
		src, err := r.srcs[i].Eval(ectx, this)
		if err != nil {
			return nil, nil, err
		}
		dst, err := r.dsts[i].Eval(ectx, this)
		if err != nil {
			return nil, nil, err
		}
		if err := CheckRenameField(src, dst); err != nil {
			return nil, nil, err
		}
		srcs = append(srcs, src)
		dsts = append(dsts, dst)
	}
	return srcs, dsts, nil
}

func (r *Renamer) computeType(typ *zed.TypeRecord, srcs, dsts field.List) (*zed.TypeRecord, error) {
	for k, dst := range dsts {
		var err error
		typ, err = r.dstType(typ, srcs[k], dst)
		if err != nil {
			return nil, err
		}
	}
	return typ, nil
}

func (r *Renamer) dstType(typ *zed.TypeRecord, src, dst field.Path) (*zed.TypeRecord, error) {
	i, ok := typ.IndexOfField(src[0])
	if !ok {
		return typ, nil
	}
	var innerType zed.Type
	if len(src) > 1 {
		recType, ok := typ.Fields[i].Type.(*zed.TypeRecord)
		if !ok {
			return typ, nil
		}
		typ, err := r.dstType(recType, src[1:], dst[1:])
		if err != nil {
			return nil, err
		}
		innerType = typ
	} else {
		innerType = typ.Fields[i].Type
	}
	fields := slices.Clone(typ.Fields)
	fields[i] = zed.NewField(dst[0], innerType)
	typ, err := r.zctx.LookupTypeRecord(fields)
	if err != nil {
		var dferr *zed.DuplicateFieldError
		if errors.As(err, &dferr) {
			return nil, err
		}
		panic(err)
	}
	return typ, nil
}

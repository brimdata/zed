package rename

import (
	"fmt"

	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
)

// Rename renames one or more fields in a record. A field can only be
// renamed within its own record. For example id.orig_h can be
// renamed to id.src, but it cannot be renamed to src. Renames are
// applied left to right; each rename observes the effect of all
// renames that preceded it.
type Proc struct {
	pctx   *proc.Context
	parent proc.Interface
	// For the dst field name, we just store the leaf name since the
	// src path and the dst path are the same and only differ in the leaf name.
	dsts    []field.Static
	srcs    []field.Static
	typeMap map[int]*zng.TypeRecord
}

func New(pctx *proc.Context, parent proc.Interface, srcs, dsts []field.Static) (*Proc, error) {
	return &Proc{
		pctx:    pctx,
		parent:  parent,
		srcs:    srcs,
		dsts:    dsts,
		typeMap: make(map[int]*zng.TypeRecord),
	}, nil
}

func (p *Proc) dstType(typ *zng.TypeRecord, src, dst field.Static) (*zng.TypeRecord, error) {
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
		innerType, err = p.dstType(recType, src[1:], dst[1:])
		if err != nil {
			return nil, err
		}
	} else {
		innerType = typ.Columns[c].Type
	}
	newcols := make([]zng.Column, len(typ.Columns))
	copy(newcols, typ.Columns)
	newcols[c] = zng.Column{Name: dst[0], Type: innerType}
	return p.pctx.TypeContext.LookupTypeRecord(newcols)
}

func (p *Proc) computeType(typ *zng.TypeRecord) (*zng.TypeRecord, error) {
	var err error
	for k, dst := range p.dsts {
		typ, err = p.dstType(typ, p.srcs[k], dst)
		if err != nil {
			return nil, err
		}
	}
	return typ, nil
}

func (p *Proc) Pull() (zbuf.Batch, error) {
	batch, err := p.parent.Pull()
	if proc.EOS(batch, err) {
		return nil, err
	}
	recs := make([]*zng.Record, 0, batch.Length())
	for k := 0; k < batch.Length(); k++ {
		in := batch.Index(k)
		id := in.Type.ID()
		if _, ok := p.typeMap[id]; !ok {
			typ, err := p.computeType(in.Type)
			if err != nil {
				return nil, fmt.Errorf("rename: %w", err)
			}
			p.typeMap[id] = typ
		}
		out := in.Keep()
		recs = append(recs, zng.NewRecord(p.typeMap[id], out.Raw))
	}
	batch.Unref()
	return zbuf.Array(recs), nil
}

func (p *Proc) Done() {
	p.parent.Done()
}

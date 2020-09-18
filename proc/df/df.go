package df

import (
	"errors"

	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

var MemMaxBytes = 128 * 1024 * 1024

type Proc struct {
	pctx   *proc.Context
	parent proc.Interface
	recs   []*zng.Record
	done   bool
}

func New(pctx *proc.Context, parent proc.Interface) (*Proc, error) {
	return &Proc{
		pctx:   pctx,
		parent: parent,
	}, nil
}

func (p *Proc) Pull() (zbuf.Batch, error) {
	if p.done {
		return nil, nil
	}
	var nbytes int
	var types resolver.Slice
	for {
		batch, err := p.parent.Pull()
		if err != nil {
			return nil, err
		}
		if batch == nil {
			p.done = true
			return p.finish(types, p.recs)
		}
		l := batch.Length()
		for i := 0; i < l; i++ {
			rec := batch.Index(i)
			id := rec.Type.ID()
			typ := types.Lookup(id)
			if typ == nil {
				types.Enter(id, rec.Type)
			}
			nbytes += len(rec.Raw)
			// We're keeping records owned by batch so don't call Unref.
			p.recs = append(p.recs, rec)
		}
		if nbytes >= MemMaxBytes {
			return nil, errors.New("df processor exceeded memory limit")
		}
	}
}

func (p *Proc) Done() {
	p.done = true
	p.parent.Done()
}

type slot struct {
	zv        zcode.Bytes
	container bool
}

func (p *Proc) finish(types resolver.Slice, recs []*zng.Record) (zbuf.Batch, error) {
	uber := newSchema()
	// positionsByID provides a map from a type ID to a slice of integers
	// that represent the column position in the uber schema for each column
	// of the input record type.
	positionsByID := make([][]int, len(types))
	for _, typ := range types {
		if typ != nil {
			id := typ.ID()
			positionsByID[id] = uber.mixin(typ)
		}
	}
	uberType, err := p.pctx.TypeContext.LookupTypeRecord(uber.columns)
	if err != nil {
		return nil, err
	}
	slots := make([]slot, len(uberType.Columns))
	for k := range slots {
		slots[k].container = zng.IsContainerType(uberType.Columns[k].Type)
	}
	var out []*zng.Record
	for _, rec := range recs {
		for k := range slots {
			slots[k].zv = nil
		}
		positions := positionsByID[rec.Type.ID()]
		it := zcode.Iter(rec.Raw)
		for _, pos := range positions {
			zv, _, err := it.Next()
			if err != nil {
				return nil, err
			}
			slots[pos].zv = zv
		}
		if !it.Done() {
			return nil, errors.New("column mismatch in df proc")
		}
		uberRec := splice(uberType, slots)
		out = append(out, uberRec)
	}
	return zbuf.Array(out), nil
}

func splice(typ *zng.TypeRecord, slots []slot) *zng.Record {
	var out zcode.Bytes
	for _, s := range slots {
		if s.container {
			out = zcode.AppendContainer(out, s.zv)
		} else {
			out = zcode.AppendPrimitive(out, s.zv)
		}
	}
	return zng.NewRecord(typ, out)
}

package fuse

import (
	"errors"
	"sync"

	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/proc/spill"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

var MemMaxBytes = 128 * 1024 * 1024
var BatchSize = 100

type Proc struct {
	pctx     *proc.Context
	parent   proc.Interface
	spiller  *spill.File
	slotByID [][]int
	slots    []slot
	uberType *zng.TypeRecord
	once     sync.Once
	resultCh chan proc.Result
	nbytes   int
	recs     []*zng.Record
	types    resolver.Slice
}

func New(pctx *proc.Context, parent proc.Interface) (*Proc, error) {
	return &Proc{
		pctx:     pctx,
		parent:   parent,
		resultCh: make(chan proc.Result),
	}, nil
}

func (p *Proc) Pull() (zbuf.Batch, error) {
	p.once.Do(func() { go p.run() })
	if r, ok := <-p.resultCh; ok {
		return r.Batch, r.Err
	}
	return nil, p.pctx.Err()
}

func (p *Proc) run() {
	if err := p.pullInput(); err != nil {
		p.shutdown(err)
		return
	}
	p.shutdown(p.pushOutput())
}

func (p *Proc) pullInput() error {
	for {
		if err := p.pctx.Err(); err != nil {
			return err
		}
		batch, err := p.parent.Pull()
		if err != nil {
			return err
		}
		if batch == nil {
			return p.finish()
		}
		if err := p.writeBatch(batch); err != nil {
			return err
		}
	}
}

func (p *Proc) writeBatch(batch zbuf.Batch) error {
	defer batch.Unref()
	l := batch.Length()
	for i := 0; i < l; i++ {
		rec := batch.Index(i)
		id := rec.Type.ID()
		typ := p.types.Lookup(id)
		if typ == nil {
			p.types.Enter(id, rec.Type)
		}
		if p.spiller != nil {
			if err := p.spiller.Write(rec); err != nil {
				return err
			}
		} else {
			if err := p.stash(rec); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *Proc) stash(rec *zng.Record) error {
	p.nbytes += len(rec.Raw)
	if p.nbytes >= MemMaxBytes {
		var err error
		p.spiller, err = spill.NewTempFile()
		if err != nil {
			return err
		}
		if err := p.spiller.Write(rec); err != nil {
			return err
		}
		for _, rec := range p.recs {
			if err := p.spiller.Write(rec); err != nil {
				return err
			}
		}
		p.recs = nil
		return nil
	}
	rec = rec.Keep()
	p.recs = append(p.recs, rec)
	return nil
}

func (p *Proc) pushOutput() error {
	var reader zbuf.Reader
	if p.spiller != nil {
		if err := p.spiller.Rewind(p.pctx.TypeContext); err != nil {
			return err
		}
		reader = p.spiller
	} else {
		reader = zbuf.Array(p.recs).NewReader()
	}
	for {
		if err := p.pctx.Err(); err != nil {
			return err
		}
		batch, err := p.nextBatch(reader)
		if err != nil || batch == nil {
			return err
		}
		p.sendResult(batch, nil)
	}
}

func (p *Proc) sendResult(b zbuf.Batch, err error) {
	select {
	case p.resultCh <- proc.Result{Batch: b, Err: err}:
	case <-p.pctx.Done():
	}
}

func (p *Proc) shutdown(err error) {
	if p.spiller != nil {
		p.spiller.CloseAndRemove()
	}
	p.sendResult(nil, err)
	close(p.resultCh)
}

func (p *Proc) Done() {
	p.parent.Done()
}

type slot struct {
	zv        zcode.Bytes
	container bool
}

func (p *Proc) finish() error {
	uber := newSchema()
	// positionsByID provides a map from a type ID to a slice of integers
	// that represent the column position in the uber schema for each column
	// of the input record type.
	p.slotByID = make([][]int, len(p.types))
	for _, typ := range p.types {
		if typ != nil {
			id := typ.ID()
			p.slotByID[id] = uber.mixin(typ)
		}
	}
	var err error
	p.uberType, err = p.pctx.TypeContext.LookupTypeRecord(uber.columns)
	if err != nil {
		return err
	}
	p.slots = make([]slot, len(p.uberType.Columns))
	for k := range p.slots {
		p.slots[k].container = zng.IsContainerType(p.uberType.Columns[k].Type)
	}
	return nil
}

func (p *Proc) nextBatch(reader zbuf.Reader) (zbuf.Batch, error) {
	var out []*zng.Record
	for len(out) < BatchSize {
		rec, err := reader.Read()
		if err != nil {
			return nil, err
		}
		if rec == nil {
			break
		}
		for k := range p.slots {
			p.slots[k].zv = nil
		}
		slotList := p.slotByID[rec.Type.ID()]
		it := zcode.Iter(rec.Raw)
		for _, slot := range slotList {
			zv, _, err := it.Next()
			if err != nil {
				return nil, err
			}
			p.slots[slot].zv = zv
		}
		if !it.Done() {
			return nil, errors.New("column mismatch in fuse processor")
		}
		uberRec := splice(p.uberType, p.slots)
		out = append(out, uberRec)
	}
	if out == nil {
		return nil, nil
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

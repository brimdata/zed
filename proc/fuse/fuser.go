package fuse

import (
	"errors"

	"github.com/brimsec/zq/proc/spill"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

// Fuser buffers records written to it, assembling from them a unified schema of
// fields and types.  Fuser then transforms those records to the unified schema
// as they are read back from it.
type Fuser struct {
	zctx        *resolver.Context
	memMaxBytes int

	nbytes   int
	recs     []*zng.Record
	slotByID [][]int
	slots    []slot
	spiller  *spill.File
	types    resolver.Slice
	uberType *zng.TypeRecord
}

type slot struct {
	zv        zcode.Bytes
	container bool
}

// NewFuser returns a new Fuser.  The Fuser buffers records in memory until
// their cumulative size (measured in zcode.Bytes length) exceeds memMaxBytes,
// at which point it buffers them in a temporary file.
func NewFuser(zctx *resolver.Context, memMaxBytes int) *Fuser {
	return &Fuser{zctx: zctx, memMaxBytes: memMaxBytes}
}

// Close removes the receiver's temporary file if it created one.
func (f *Fuser) Close() error {
	if f.spiller != nil {
		return f.spiller.CloseAndRemove()
	}
	return nil
}

// Write buffers rec. If called after Read, Write panics.
func (f *Fuser) Write(rec *zng.Record) error {
	if f.finished() {
		panic("fuser: write after read")
	}
	id := rec.Type.ID()
	typ := f.types.Lookup(id)
	if typ == nil {
		f.types.Enter(id, rec.Type)
	}
	if f.spiller != nil {
		return f.spiller.Write(rec)
	}
	return f.stash(rec)
}

func (f *Fuser) stash(rec *zng.Record) error {
	f.nbytes += len(rec.Raw)
	if f.nbytes >= f.memMaxBytes {
		var err error
		f.spiller, err = spill.NewTempFile()
		if err != nil {
			return err
		}
		for _, rec := range f.recs {
			if err := f.spiller.Write(rec); err != nil {
				return err
			}
		}
		f.recs = nil
		return f.spiller.Write(rec)
	}
	rec = rec.Keep()
	f.recs = append(f.recs, rec)
	return nil
}

func (f *Fuser) finished() bool {
	return f.slotByID != nil
}

func (f *Fuser) finish() error {
	uber := newSchema()
	// slotByID provides a map from a type ID to a slice of integers
	// that represent the column position in the uber schema for each column
	// of the input record type.
	f.slotByID = make([][]int, len(f.types))
	for _, typ := range f.types {
		if typ != nil {
			id := typ.ID()
			f.slotByID[id] = uber.mixin(zng.AliasedType(typ).(*zng.TypeRecord))
		}
	}
	var err error
	f.uberType, err = f.zctx.LookupTypeRecord(uber.columns)
	if err != nil {
		return err
	}
	f.slots = make([]slot, len(f.uberType.Columns))
	for k := range f.slots {
		f.slots[k].container = zng.IsContainerType(f.uberType.Columns[k].Type)
	}
	if f.spiller != nil {
		return f.spiller.Rewind(f.zctx)
	}
	return nil
}

// Read returns the next buffered record after transforming it to the unified
// schema.
func (f *Fuser) Read() (*zng.Record, error) {
	if !f.finished() {
		if err := f.finish(); err != nil {
			return nil, err
		}
	}
	rec, err := f.next()
	if rec == nil || err != nil {
		return nil, err
	}
	for k := range f.slots {
		f.slots[k].zv = nil
	}
	slotList := f.slotByID[rec.Type.ID()]
	it := rec.Raw.Iter()
	for _, slot := range slotList {
		zv, _, err := it.Next()
		if err != nil {
			return nil, err
		}
		f.slots[slot].zv = zv
	}
	if !it.Done() {
		return nil, errors.New("column mismatch in fuse processor")
	}
	return splice(f.uberType, f.slots), nil
}

func (f *Fuser) next() (*zng.Record, error) {
	if f.spiller != nil {
		return f.spiller.Read()
	}
	var rec *zng.Record
	if len(f.recs) > 0 {
		rec = f.recs[0]
		f.recs = f.recs[1:]
	}
	return rec, nil

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

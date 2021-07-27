package fuse

import (
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/expr/agg"
	"github.com/brimdata/zed/proc/spill"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

// Fuser buffers records written to it, assembling from them a unified schema of
// fields and types.  Fuser then transforms those records to the unified schema
// as they are read back from it.
type Fuser struct {
	zctx        *zson.Context
	memMaxBytes int

	nbytes  int
	recs    []*zng.Record
	spiller *spill.File

	types      map[zng.Type]struct{}
	uberSchema *agg.Schema
	shaper     *expr.ConstShaper
}

// NewFuser returns a new Fuser.  The Fuser buffers records in memory until
// their cumulative size (measured in zcode.Bytes length) exceeds memMaxBytes,
// at which point it buffers them in a temporary file.
func NewFuser(zctx *zson.Context, memMaxBytes int) *Fuser {
	s, err := agg.NewSchema(zctx)
	if err != nil {
		panic(err)
	}
	return &Fuser{
		zctx:        zctx,
		memMaxBytes: memMaxBytes,
		types:       make(map[zng.Type]struct{}),
		uberSchema:  s,
	}
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
	if f.shaper != nil {
		panic("fuser: write after read")
	}
	if _, ok := f.types[rec.Type]; !ok {
		f.types[rec.Type] = struct{}{}
		if err := f.uberSchema.Mixin(zng.TypeRecordOf(rec.Type)); err != nil {
			return err
		}
	}
	if f.spiller != nil {
		return f.spiller.Write(rec)
	}
	return f.stash(rec)
}

func (f *Fuser) stash(rec *zng.Record) error {
	f.nbytes += len(rec.Bytes)
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

// Read returns the next buffered record after transforming it to the unified
// schema.
func (f *Fuser) Read() (*zng.Record, error) {
	if f.shaper == nil {
		f.shaper = expr.NewConstShaper(f.zctx, &expr.RootRecord{}, f.uberSchema.Type, expr.Cast|expr.Fill|expr.Order)
		if f.spiller != nil {
			if err := f.spiller.Rewind(f.zctx); err != nil {
				return nil, err
			}
		}
	}
	rec, err := f.next()
	if rec == nil || err != nil {
		return nil, err
	}
	return f.shaper.Apply(rec)
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

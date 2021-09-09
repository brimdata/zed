package spill

import (
	"context"

	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

type peeker struct {
	*File
	nextRecord *zng.Record
	ordinal    int
}

func newPeeker(ctx context.Context, filename string, ordinal int, recs []*zng.Record, zctx *zson.Context) (*peeker, error) {
	f, err := NewFileWithPath(filename, zctx)
	if err != nil {
		return nil, err
	}
	arr := zbuf.Array(recs)
	if err := zio.CopyWithContext(ctx, f, &arr); err != nil {
		f.CloseAndRemove()
		return nil, err
	}
	if err := f.Rewind(zctx); err != nil {
		f.CloseAndRemove()
		return nil, err
	}
	first, err := f.Read()
	if err != nil {
		f.CloseAndRemove()
		return nil, err
	}
	return &peeker{f, first, ordinal}, nil
}

// read is like Read but returns eof at the last record so a MergeSort can
// do its heap management a bit more easily.
func (p *peeker) read() (*zng.Record, bool, error) {
	rec := p.nextRecord
	if rec != nil {
		rec = rec.Keep()
	}
	var err error
	p.nextRecord, err = p.Read()
	eof := p.nextRecord == nil && err == nil
	return rec, eof, err
}

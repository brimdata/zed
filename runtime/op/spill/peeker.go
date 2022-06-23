package spill

import (
	"context"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
)

type peeker struct {
	*File
	nextRecord *zed.Value
	ordinal    int
}

func newPeeker(ctx context.Context, zctx *zed.Context, filename string, ordinal int, arr *zbuf.Array) (*peeker, error) {
	f, err := NewFileWithPath(filename)
	if err != nil {
		return nil, err
	}
	if err := zio.CopyWithContext(ctx, f, arr); err != nil {
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
func (p *peeker) read() (*zed.Value, bool, error) {
	rec := p.nextRecord
	if rec != nil {
		rec = rec.Copy()
	}
	var err error
	p.nextRecord, err = p.Read()
	eof := p.nextRecord == nil && err == nil
	return rec, eof, err
}

package spill

import (
	"context"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio"
)

type peeker struct {
	*File
	arena      *zed.Arena
	nextRecord *zed.Value
	ordinal    int
}

func newPeeker(ctx context.Context, zctx *zed.Context, filename string, ordinal int, zr zio.Reader) (*peeker, error) {
	f, err := NewFileWithPath(filename)
	if err != nil {
		return nil, err
	}
	if err := zio.CopyWithContext(ctx, f, zr); err != nil {
		f.CloseAndRemove()
		return nil, err
	}
	if err := f.Rewind(zctx); err != nil {
		f.CloseAndRemove()
		return nil, err
	}
	arena := zed.NewArena(zctx)
	first, err := f.Read(arena)
	if err != nil {
		f.CloseAndRemove()
		return nil, err
	}
	return &peeker{f, arena, first, ordinal}, nil
}

// read is like Read but returns eof at the last record so a MergeSort can
// do its heap management a bit more easily.
func (p *peeker) read() (*zed.Value, bool, error) {
	rec := p.nextRecord
	if rec != nil {
		rec = rec.Copy().Ptr()
	}
	var err error
	p.nextRecord, err = p.Read(p.arena)
	eof := p.nextRecord == nil && err == nil
	return rec, eof, err
}

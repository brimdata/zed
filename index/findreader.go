package index

import (
	"context"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zio/zngio"
)

// FinderReader is a zio.ReadCloser version of Finder that streams back all
// records in a microindex that match the provided key record.
type FinderReader struct {
	compare keyCompareFn
	finder  *Finder
	inputs  []string
	reader  *zngio.Reader
}

// NewFinderReader returns a new FinderReader for the microindex at uri and the
// key record in inputs.  See Finder.ParseKeys for an explanation of how the key
// record is constructed from inputs.
//
// It is the caller's responsibility to call Close on the FinderReader when
// done.
func NewFinderReader(ctx context.Context, zctx *zed.Context, engine storage.Engine, uri *storage.URI, inputs ...string) (*FinderReader, error) {
	finder, err := NewFinder(ctx, zctx, engine, uri)
	if err != nil {
		return nil, err
	}
	return &FinderReader{finder: finder, inputs: inputs}, nil
}

func (f *FinderReader) init() error {
	kvs, err := f.finder.ParseKeys(f.inputs...)
	if err != nil {
		return err
	}
	f.compare = compareFn(f.finder.zctx, kvs)
	if err != nil {
		return err
	}
	f.reader, err = f.finder.search(f.compare)
	return err
}

func (f *FinderReader) Read() (*zed.Value, error) {
	if f.finder.IsEmpty() {
		return nil, nil
	}
	if f.compare == nil {
		if err := f.init(); err != nil {
			return nil, err
		}
	}
	return lookup(f.reader, f.compare, f.finder.meta.Order, EQL)
}

func (f *FinderReader) Close() error {
	err := f.finder.Close()
	if f.reader != nil {
		if err2 := f.reader.Close(); err == nil {
			err = err2
		}
	}
	return err
}

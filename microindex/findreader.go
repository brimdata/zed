package microindex

import (
	"context"

	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zng/resolver"
)

// FinderReader is zbuf.Reader version of Finder that streams back all records
// in a microindex that match the provided key Record.
type FinderReader struct {
	compare expr.KeyCompareFn
	finder  *Finder
	inputs  []string
	reader  zbuf.Reader
}

func NewFinderReader(ctx context.Context, zctx *resolver.Context, uri iosrc.URI, inputs ...string) (*FinderReader, error) {
	finder, err := NewFinder(ctx, zctx, uri)
	if err != nil {
		return nil, err
	}
	return &FinderReader{finder: finder, inputs: inputs}, nil
}

func (f *FinderReader) init() error {
	keys, err := f.finder.ParseKeys(f.inputs...)
	if err != nil {
		return err
	}
	f.compare, err = expr.NewKeyCompareFn(keys)
	if err != nil {
		return err
	}
	f.reader, err = f.finder.search(f.compare)
	return err
}

func (f *FinderReader) Read() (*zng.Record, error) {
	if f.finder.IsEmpty() {
		return nil, nil
	}
	if f.compare == nil {
		if err := f.init(); err != nil {
			return nil, err
		}
	}
	return lookup(f.reader, f.compare, f.finder.trailer.Order, eql)
}

func (f *FinderReader) Close() error {
	return f.finder.Close()
}

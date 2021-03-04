package microindex

import (
	"context"

	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

// FinderReader is zbuf.Reader version of Finder that streams back all records
// in a microindex that match the provided key Record.
type FinderReader struct {
	compare expr.KeyCompareFn
	finder  *Finder
	reader  zbuf.Reader
}

func NewFinderReader(ctx context.Context, zctx *resolver.Context, uri iosrc.URI, inputs ...string) (*FinderReader, error) {
	finder, err := NewFinder(ctx, zctx, uri)
	if err != nil {
		return nil, err
	}
	keys, err := finder.ParseKeys(inputs...)
	if err != nil {
		finder.Close()
		return nil, err
	}
	compare, err := expr.NewKeyCompareFn(keys)
	if err != nil {
		finder.Close()
		return nil, err
	}
	reader, err := finder.search(compare)
	if err != nil {
		finder.Close()
		return nil, err
	}
	return &FinderReader{compare: compare, finder: finder, reader: reader}, nil
}

func (f *FinderReader) Read() (*zng.Record, error) {
	if f.finder.IsEmpty() {
		return nil, nil
	}
	return lookup(f.reader, f.compare, f.finder.trailer.Order, eql)
}

func (f *FinderReader) Close() error {
	return f.finder.Close()
}

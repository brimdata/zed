package index

import (
	"context"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/zio"
)

// FinderReader is zio.Reader version of Finder that streams back all records
// in a microindex that match the provided key Record.
type FinderReader struct {
	finder     *Finder
	spanFilter *expr.SpanFilter
	valFilter  expr.Evaluator
	reader     zio.Reader
}

func NewFinderReader(ctx context.Context, zctx *zed.Context, engine storage.Engine, uri *storage.URI, filter dag.Expr) (*FinderReader, error) {
	finder, err := NewFinder(ctx, zctx, engine, uri)
	if err != nil {
		return nil, err
	}
	spanFilter, valFilter, err := compileFilter(filter, finder.meta.Keys[0], finder.meta.Order)
	if err != nil {
		return nil, err
	}

	return &FinderReader{
		finder:     finder,
		spanFilter: spanFilter,
		valFilter:  valFilter,
	}, nil
}

func (f *FinderReader) init() error {
	var err error
	f.reader, err = f.finder.search(f.spanFilter)
	return err
}

func (f *FinderReader) Read() (*zed.Value, error) {
	if f.finder.IsEmpty() {
		return nil, nil
	}
	if f.reader == nil {
		if err := f.init(); err != nil {
			return nil, err
		}
	}
	return lookup(f.reader, f.valFilter)
}

func (f *FinderReader) Close() error {
	return f.finder.Close()
}

package anyio

import (
	"context"
	"io"

	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
)

// Open uses engine to open path for reading.  path is a local file path or a
// URI whose scheme is understood by engine.
func Open(ctx context.Context, zctx *zson.Context, engine storage.Engine, path string, opts ReaderOpts) (*zbuf.File, error) {
	uri, err := storage.ParseURI(path)
	if err != nil {
		return nil, err
	}
	f, err := engine.Get(ctx, uri)
	if err != nil {
		return nil, err
	}
	return NewFile(zctx, f, path, opts)
}

func NewFile(zctx *zson.Context, rc io.ReadCloser, path string, opts ReaderOpts) (*zbuf.File, error) {
	var err error
	r := io.Reader(rc)
	if opts.Format != "parquet" && opts.Format != "zst" {
		r = GzipReader(rc)
	}
	var zr zio.Reader
	if opts.Format == "" || opts.Format == "auto" {
		zr, err = NewReaderWithOpts(r, zctx, opts)
	} else {
		zr, err = lookupReader(r, zctx, opts)
	}
	if err != nil {
		return nil, err
	}

	return zbuf.NewFile(zr, rc, path), nil
}

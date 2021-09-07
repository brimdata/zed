package anyio

import (
	"context"
	"io"

	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
)

// OpenFile creates and returns zbuf.File for the indicated "path",
// which can be a local file path, a local directory path, or an S3
// URL. If the path is neither of these or can't otherwise be opened,
// an error is returned.
func OpenFile(zctx *zson.Context, engine storage.Engine, path string, opts ReaderOpts) (*zbuf.File, error) {
	return OpenFileWithContext(context.Background(), zctx, engine, path, opts)
}

func OpenFileWithContext(ctx context.Context, zctx *zson.Context, engine storage.Engine, path string, opts ReaderOpts) (*zbuf.File, error) {
	uri, err := storage.ParseURI(path)
	if err != nil {
		return nil, err
	}
	f, err := engine.Get(ctx, uri)
	if err != nil {
		return nil, err
	}
	return OpenFromNamedReadCloser(zctx, f, path, opts)
}

func OpenFromNamedReadCloser(zctx *zson.Context, rc io.ReadCloser, path string, opts ReaderOpts) (*zbuf.File, error) {
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

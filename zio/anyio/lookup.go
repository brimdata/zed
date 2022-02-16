package anyio

import (
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/csvio"
	"github.com/brimdata/zed/zio/jsonio"
	"github.com/brimdata/zed/zio/parquetio"
	"github.com/brimdata/zed/zio/zeekio"
	"github.com/brimdata/zed/zio/zjsonio"
	"github.com/brimdata/zed/zio/zng21io"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/brimdata/zed/zio/zstio"
)

func lookupReader(r io.Reader, zctx *zed.Context, opts ReaderOpts) (zio.ReadCloser, error) {
	switch opts.Format {
	case "csv":
		return zio.NopReadCloser(csvio.NewReader(r, zctx)), nil
	case "zeek":
		return zio.NopReadCloser(zeekio.NewReader(r, zctx)), nil
	case "json":
		return zio.NopReadCloser(jsonio.NewReader(r, zctx)), nil
	case "zjson":
		return zio.NopReadCloser(zjsonio.NewReader(r, zctx)), nil
	case "zng":
		return zngio.NewReaderWithOpts(r, zctx, opts.ZNG), nil
	case "zng21":
		return zio.NopReadCloser(zng21io.NewReaderWithOpts(r, zctx, opts.ZNG)), nil
	case "zson":
		return zio.NopReadCloser(zsonio.NewReader(r, zctx)), nil
	case "zst":
		zr, err := zstio.NewReader(r, zctx)
		if err != nil {
			return nil, err
		}
		return zio.NopReadCloser(zr), nil
	case "parquet":
		zr, err := parquetio.NewReader(r, zctx)
		if err != nil {
			return nil, err
		}
		return zio.NopReadCloser(zr), nil
	}
	return nil, fmt.Errorf("no such format: \"%s\"", opts.Format)
}

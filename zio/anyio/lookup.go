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

func lookupReader(zctx *zed.Context, r io.Reader, opts ReaderOpts) (zio.ReadCloser, error) {
	switch opts.Format {
	case "csv":
		return zio.NopReadCloser(csvio.NewReader(zctx, r)), nil
	case "zeek":
		return zio.NopReadCloser(zeekio.NewReader(zctx, r)), nil
	case "json":
		return zio.NopReadCloser(jsonio.NewReader(zctx, r)), nil
	case "zjson":
		return zio.NopReadCloser(zjsonio.NewReader(zctx, r)), nil
	case "zng":
		return zngio.NewReaderWithOpts(zctx, r, opts.ZNG), nil
	case "zng21":
		return zio.NopReadCloser(zng21io.NewReaderWithOpts(zctx, r, opts.ZNG)), nil
	case "zson":
		return zio.NopReadCloser(zsonio.NewReader(zctx, r)), nil
	case "zst":
		zr, err := zstio.NewReader(zctx, r)
		if err != nil {
			return nil, err
		}
		return zio.NopReadCloser(zr), nil
	case "parquet":
		zr, err := parquetio.NewReader(zctx, r)
		if err != nil {
			return nil, err
		}
		return zio.NopReadCloser(zr), nil
	}
	return nil, fmt.Errorf("no such format: \"%s\"", opts.Format)
}

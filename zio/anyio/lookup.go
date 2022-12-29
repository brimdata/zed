package anyio

import (
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/arrowio"
	"github.com/brimdata/zed/zio/csvio"
	"github.com/brimdata/zed/zio/jsonio"
	"github.com/brimdata/zed/zio/lineio"
	"github.com/brimdata/zed/zio/parquetio"
	"github.com/brimdata/zed/zio/vngio"
	"github.com/brimdata/zed/zio/zeekio"
	"github.com/brimdata/zed/zio/zjsonio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zio/zsonio"
)

func lookupReader(zctx *zed.Context, r io.Reader, opts ReaderOpts) (zio.ReadCloser, error) {
	switch opts.Format {
	case "arrows":
		return arrowio.NewReader(zctx, r)
	case "csv":
		return zio.NopReadCloser(csvio.NewReader(zctx, r)), nil
	case "line":
		return zio.NopReadCloser(lineio.NewReader(r)), nil
	case "json":
		return zio.NopReadCloser(jsonio.NewReader(zctx, r)), nil
	case "parquet":
		zr, err := parquetio.NewReader(zctx, r)
		if err != nil {
			return nil, err
		}
		return zio.NopReadCloser(zr), nil
	case "vng":
		zr, err := vngio.NewReader(zctx, r)
		if err != nil {
			return nil, err
		}
		return zio.NopReadCloser(zr), nil
	case "zeek":
		return zio.NopReadCloser(zeekio.NewReader(zctx, r)), nil
	case "zjson":
		return zio.NopReadCloser(zjsonio.NewReader(zctx, r)), nil
	case "zng":
		return zngio.NewReaderWithOpts(zctx, r, opts.ZNG), nil
	case "zson":
		return zio.NopReadCloser(zsonio.NewReader(zctx, r)), nil
	}
	return nil, fmt.Errorf("no such format: \"%s\"", opts.Format)
}

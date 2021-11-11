package anyio

import (
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/csvio"
	"github.com/brimdata/zed/zio/jsonio"
	"github.com/brimdata/zed/zio/lakeio"
	"github.com/brimdata/zed/zio/ndjsonio"
	"github.com/brimdata/zed/zio/parquetio"
	"github.com/brimdata/zed/zio/tableio"
	"github.com/brimdata/zed/zio/textio"
	"github.com/brimdata/zed/zio/zeekio"
	"github.com/brimdata/zed/zio/zjsonio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/brimdata/zed/zio/zstio"
)

type WriterOpts struct {
	Format string
	UTF8   bool
	JSON   jsonio.WriterOpts
	Lake   lakeio.WriterOpts
	Text   textio.WriterOpts
	Zng    zngio.WriterOpts
	ZSON   zsonio.WriterOpts
	Zst    zstio.WriterOpts
}

func NewWriter(w io.WriteCloser, opts WriterOpts) (zio.WriteCloser, error) {
	switch opts.Format {
	case "null":
		return &nullWriter{}, nil
	case "zng":
		return zngio.NewWriter(w, opts.Zng), nil
	case "zeek":
		return zeekio.NewWriter(w, opts.UTF8), nil
	case "ndjson":
		return ndjsonio.NewWriter(w), nil
	case "json":
		return jsonio.NewWriter(w, opts.JSON), nil
	case "zjson":
		return zjsonio.NewWriter(w), nil
	case "zson", "":
		return zsonio.NewWriter(w, opts.ZSON), nil
	case "zst":
		return zstio.NewWriter(w, opts.Zst)
	case "text":
		return textio.NewWriter(w, opts.UTF8, opts.Text), nil
	case "table":
		return tableio.NewWriter(w, opts.UTF8), nil
	case "csv":
		return csvio.NewWriter(w), nil
	case "parquet":
		return parquetio.NewWriter(w), nil
	case "lake":
		return lakeio.NewWriter(w, opts.Lake), nil
	default:
		return nil, fmt.Errorf("unknown format: %s", opts.Format)
	}
}

type nullWriter struct{}

func (*nullWriter) Write(*zed.Value) error {
	return nil
}

func (*nullWriter) Close() error {
	return nil
}

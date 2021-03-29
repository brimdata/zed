package detector

import (
	"fmt"
	"io"

	"github.com/brimdata/zq/zbuf"
	"github.com/brimdata/zq/zio"
	"github.com/brimdata/zq/zio/csvio"
	"github.com/brimdata/zq/zio/ndjsonio"
	"github.com/brimdata/zq/zio/parquetio"
	"github.com/brimdata/zq/zio/tableio"
	"github.com/brimdata/zq/zio/textio"
	"github.com/brimdata/zq/zio/tzngio"
	"github.com/brimdata/zq/zio/zeekio"
	"github.com/brimdata/zq/zio/zjsonio"
	"github.com/brimdata/zq/zio/zngio"
	"github.com/brimdata/zq/zio/zsonio"
	"github.com/brimdata/zq/zio/zstio"
	"github.com/brimdata/zq/zng"
	"github.com/brimdata/zq/zng/resolver"
	"github.com/brimdata/zq/zson"
)

type nullWriter struct{}

func (*nullWriter) Write(*zng.Record) error {
	return nil
}

func (*nullWriter) Close() error {
	return nil
}

func LookupWriter(w io.WriteCloser, zctx *resolver.Context, opts zio.WriterOpts) (zbuf.WriteCloser, error) {
	if opts.Format == "" {
		opts.Format = "tzng"
	}
	switch opts.Format {
	case "null":
		return &nullWriter{}, nil
	case "tzng":
		return tzngio.NewWriter(w), nil
	case "zng":
		return zngio.NewWriter(w, opts.Zng), nil
	case "zeek":
		return zeekio.NewWriter(w, opts.UTF8), nil
	case "ndjson":
		return ndjsonio.NewWriter(w), nil
	case "zjson":
		return zjsonio.NewWriter(w), nil
	case "zson":
		return zsonio.NewWriter(w, opts.ZSON), nil
	case "zst":
		return zstio.NewWriter(w, opts.Zst)
	case "text":
		return textio.NewWriter(w, opts.UTF8, opts.Text, opts.EpochDates), nil
	case "table":
		return tableio.NewWriter(w, opts.UTF8, opts.EpochDates), nil
	case "csv":
		return csvio.NewWriter(w, zctx, csvio.WriterOpts{
			EpochDates: opts.EpochDates,
			Fuse:       opts.CSVFuse,
			UTF8:       opts.UTF8,
		}), nil
	case "parquet":
		return parquetio.NewWriter(w), nil
	default:
		return nil, fmt.Errorf("unknown format: %s", opts.Format)
	}
}

func lookupReader(r io.Reader, zctx *resolver.Context, path string, opts zio.ReaderOpts) (zbuf.Reader, error) {
	switch opts.Format {
	case "csv":
		return csvio.NewReader(r, zctx.Context), nil
	case "tzng":
		return tzngio.NewReader(r, zctx), nil
	case "zeek":
		return zeekio.NewReader(r, zctx)
	case "ndjson":
		return ndjsonio.NewReader(r, zctx, opts.JSON, path)
	case "zjson":
		return zjsonio.NewReader(r, zctx), nil
	case "zng":
		return zngio.NewReaderWithOpts(r, zctx, opts.Zng), nil
	case "zson":
		return zson.NewReader(r, zctx.Context), nil
	case "zst":
		return zstio.NewReader(r, zctx)
	case "parquet":
		return parquetio.NewReader(r, zctx)
	}
	return nil, fmt.Errorf("no such format: \"%s\"", opts.Format)
}

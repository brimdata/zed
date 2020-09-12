package detector

import (
	"fmt"
	"io"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/ndjsonio"
	"github.com/brimsec/zq/zio/options"
	"github.com/brimsec/zq/zio/tableio"
	"github.com/brimsec/zq/zio/textio"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zio/zeekio"
	"github.com/brimsec/zq/zio/zjsonio"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type nullWriter struct{}

func (*nullWriter) Write(*zng.Record) error {
	return nil
}

func (*nullWriter) Close() error {
	return nil
}

func LookupWriter(w io.WriteCloser, opts options.Writer) zbuf.WriteCloser {
	if opts.Format == "" {
		opts.Format = "tzng"
	}
	switch opts.Format {
	default:
		return nil
	case "null":
		return &nullWriter{}
	case "tzng":
		return tzngio.NewWriter(w)
	case "zng":
		return zngio.NewWriter(w, opts.Zng)
	case "zeek":
		return zeekio.NewWriter(w, opts.UTF8)
	case "ndjson":
		return ndjsonio.NewWriter(w)
	case "zjson":
		return zjsonio.NewWriter(w)
	case "text":
		return textio.NewWriter(w, opts.UTF8, opts.Text)
	case "table":
		return tableio.NewWriter(w, opts.UTF8)
	}
}

func lookupReader(r io.Reader, zctx *resolver.Context, path string, opts options.Reader) (zbuf.Reader, error) {
	switch opts.Format {
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
	}
	return nil, fmt.Errorf("no such format: \"%s\"", opts.Format)
}

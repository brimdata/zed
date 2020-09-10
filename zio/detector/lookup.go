package detector

import (
	"fmt"
	"io"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/ndjsonio"
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

func LookupWriter(w io.WriteCloser, wflags *zio.WriterFlags) zbuf.WriteCloser {
	flags := *wflags
	if flags.Format == "" {
		flags.Format = "tzng"
	}
	switch flags.Format {
	default:
		return nil
	case "null":
		return &nullWriter{}
	case "tzng":
		return tzngio.NewWriter(w)
	case "zng":
		return zngio.NewWriter(w, flags)
	case "zeek":
		return zeekio.NewWriter(w, flags)
	case "ndjson":
		return ndjsonio.NewWriter(w)
	case "zjson":
		return zjsonio.NewWriter(w)
	case "text":
		return textio.NewWriter(w, flags)
	case "table":
		return tableio.NewWriter(w, flags)
	}
}

func lookupReader(r io.Reader, zctx *resolver.Context, path string, cfg OpenConfig) (zbuf.Reader, error) {
	switch cfg.Format {
	case "tzng":
		return tzngio.NewReader(r, zctx), nil
	case "zeek":
		return zeekio.NewReader(r, zctx)
	case "ndjson":
		return ndjsonio.NewReader(r, zctx, cfg.JSONTypeConfig, cfg.JSONPathRegex, path)
	case "zjson":
		return zjsonio.NewReader(r, zctx), nil
	case "zng":
		return zngio.NewReaderWithOpts(r, zctx, zngio.ReaderOpts{Check: cfg.ZngCheck}), nil
	}
	return nil, fmt.Errorf("no such reader type: \"%s\"", cfg.Format)
}

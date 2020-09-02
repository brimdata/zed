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

func LookupWriter(w io.WriteCloser, wflags *zio.WriterFlags) *zio.Writer {
	flags := *wflags
	if flags.Format == "" {
		flags.Format = "tzng"
	}
	var f zbuf.WriteFlusher
	switch flags.Format {
	default:
		return nil
	case "null":
		f = zbuf.NopFlusher(&nullWriter{})
	case "tzng":
		f = zbuf.NopFlusher(tzngio.NewWriter(w))
	case "zng":
		f = zngio.NewWriter(w, flags)
	case "zeek":
		f = zbuf.NopFlusher(zeekio.NewWriter(w, flags))
	case "ndjson":
		f = zbuf.NopFlusher(ndjsonio.NewWriter(w))
	case "zjson":
		f = zbuf.NopFlusher(zjsonio.NewWriter(w))
	case "text":
		f = zbuf.NopFlusher(textio.NewWriter(w, flags))
	case "table":
		f = tableio.NewWriter(w, flags)
	}
	return &zio.Writer{
		WriteFlusher: f,
		Closer:       w,
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

package detector

import (
	"io"

	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zio"
	"github.com/mccanne/zq/zio/bzngio"
	"github.com/mccanne/zq/zio/ndjsonio"
	"github.com/mccanne/zq/zio/tableio"
	"github.com/mccanne/zq/zio/textio"
	"github.com/mccanne/zq/zio/zeekio"
	"github.com/mccanne/zq/zio/zjsonio"
	"github.com/mccanne/zq/zio/zngio"
	"github.com/mccanne/zq/zng/resolver"
)

func LookupWriter(format string, w io.WriteCloser, optionalFlags *zio.Flags) *zio.Writer {
	var flags zio.Flags
	if optionalFlags != nil {
		flags = *optionalFlags
	}
	var f zbuf.WriteFlusher
	switch format {
	default:
		return nil
	case "zng":
		f = zbuf.NopFlusher(zngio.NewWriter(w))
	case "bzng":
		f = zbuf.NopFlusher(bzngio.NewWriter(w))
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

func LookupReader(format string, r io.Reader, zctx *resolver.Context) zbuf.Reader {
	switch format {
	case "zng", "zeek":
		return zngio.NewReader(r, zctx)
	case "ndjson":
		return ndjsonio.NewReader(r, zctx)
	case "zjson":
		return zjsonio.NewReader(r, zctx)
	case "bzng":
		return bzngio.NewReader(r, zctx)
	}
	return nil
}

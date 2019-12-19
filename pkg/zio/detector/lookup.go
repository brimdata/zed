package detector

import (
	"io"

	"github.com/mccanne/zq/pkg/zio"
	"github.com/mccanne/zq/pkg/zio/bzngio"
	"github.com/mccanne/zq/pkg/zio/ndjsonio"
	"github.com/mccanne/zq/pkg/zio/tableio"
	"github.com/mccanne/zq/pkg/zio/textio"
	"github.com/mccanne/zq/pkg/zio/zeekio"
	"github.com/mccanne/zq/pkg/zio/zjsonio"
	"github.com/mccanne/zq/pkg/zio/zngio"
	"github.com/mccanne/zq/pkg/zng"
	"github.com/mccanne/zq/pkg/zng/resolver"
)

func LookupWriter(format string, w io.WriteCloser, optionalFlags *zio.Flags) *zio.Writer {
	var flags zio.Flags
	if optionalFlags != nil {
		flags = *optionalFlags
	}
	var f zng.WriteFlusher
	switch format {
	default:
		return nil
	case "zng":
		f = zng.NopFlusher(zngio.NewWriter(w))
	case "bzng":
		f = zng.NopFlusher(bzngio.NewWriter(w))
	case "zeek":
		f = zng.NopFlusher(zeekio.NewWriter(w, flags))
	case "ndjson":
		f = zng.NopFlusher(ndjsonio.NewWriter(w))
	case "zjson":
		f = zng.NopFlusher(zjsonio.NewWriter(w))
	case "text":
		f = zng.NopFlusher(textio.NewWriter(w, flags))
	case "table":
		f = tableio.NewWriter(w, flags)
	}
	return &zio.Writer{
		WriteFlusher: f,
		Closer:       w,
	}
}

func LookupReader(format string, r io.Reader, table *resolver.Table) zng.Reader {
	switch format {
	case "zng", "zeek":
		return zngio.NewReader(r, table)
	case "ndjson":
		return ndjsonio.NewReader(r, table)
	case "zjson":
		return zjsonio.NewReader(r, table)
	case "bzng":
		return bzngio.NewReader(r, table)
	}
	return nil
}

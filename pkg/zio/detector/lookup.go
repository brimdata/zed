package detector

import (
	"io"

	"github.com/mccanne/zq/pkg/zio"
	"github.com/mccanne/zq/pkg/zio/bzsonio"
	"github.com/mccanne/zq/pkg/zio/ndjsonio"
	"github.com/mccanne/zq/pkg/zio/tableio"
	"github.com/mccanne/zq/pkg/zio/textio"
	"github.com/mccanne/zq/pkg/zio/zeekio"
	"github.com/mccanne/zq/pkg/zio/zjsonio"
	"github.com/mccanne/zq/pkg/zio/zsonio"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
)

func LookupWriter(format string, w io.WriteCloser, optionalFlags *zio.Flags) *zio.Writer {
	var flags zio.Flags
	if optionalFlags != nil {
		flags = *optionalFlags
	}
	var f zson.WriteFlusher
	switch format {
	default:
		return nil
	case "zson":
		f = zson.NopFlusher(zsonio.NewWriter(w))
	case "bzson":
		f = zson.NopFlusher(bzsonio.NewWriter(w))
	case "zeek":
		f = zson.NopFlusher(zeekio.NewWriter(w, flags))
	case "ndjson":
		f = zson.NopFlusher(ndjsonio.NewWriter(w))
	case "zjson":
		f = zson.NopFlusher(zjsonio.NewWriter(w))
	case "text":
		f = zson.NopFlusher(textio.NewWriter(w, flags))
	case "table":
		f = tableio.NewWriter(w, flags)
	}
	return &zio.Writer{
		WriteFlusher: f,
		Closer:       w,
	}
}

func LookupReader(format string, r io.Reader, table *resolver.Table) zson.Reader {
	switch format {
	case "zson", "zeek":
		return zsonio.NewReader(r, table)
	case "ndjson":
		return ndjsonio.NewReader(r, table)
	case "zjson":
		return zjsonio.NewReader(r, table)
	case "bzson":
		return bzsonio.NewReader(r, table)
	}
	return nil
}

package detector

import (
	"io"

	"github.com/mccanne/zq/pkg/zsio"
	"github.com/mccanne/zq/pkg/zsio/bzson"
	"github.com/mccanne/zq/pkg/zsio/ndjson"
	"github.com/mccanne/zq/pkg/zsio/table"
	"github.com/mccanne/zq/pkg/zsio/text"
	"github.com/mccanne/zq/pkg/zsio/zeek"
	"github.com/mccanne/zq/pkg/zsio/zjson"
	zsonio "github.com/mccanne/zq/pkg/zsio/zson"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
)

func LookupWriter(format string, w io.WriteCloser, optionalFlags *zsio.Flags) *zsio.Writer {
	var flags zsio.Flags
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
		f = zson.NopFlusher(bzson.NewWriter(w))
	case "zeek":
		f = zson.NopFlusher(zeek.NewWriter(w, flags))
	case "ndjson":
		f = zson.NopFlusher(ndjson.NewWriter(w))
	case "zjson":
		f = zson.NopFlusher(zjson.NewWriter(w))
	case "text":
		f = zson.NopFlusher(text.NewWriter(w, flags))
	case "table":
		f = table.NewWriter(w, flags)
	}
	return &zsio.Writer{
		WriteFlusher: f,
		Closer:       w,
	}
}

func LookupReader(format string, r io.Reader, table *resolver.Table) zson.Reader {
	switch format {
	case "zson", "zeek":
		return zsonio.NewReader(r, table)
	case "ndjson":
		return ndjson.NewReader(r, table)
	case "zjson":
		return zjson.NewReader(r, table)
	case "bzson":
		return bzson.NewReader(r, table)
	}
	return nil
}

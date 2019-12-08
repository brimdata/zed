package zio

import (
	"io"

	"github.com/mccanne/zq/pkg/zio/bzqio"
	"github.com/mccanne/zq/pkg/zio/ndjsonio"
	"github.com/mccanne/zq/pkg/zio/tableio"
	"github.com/mccanne/zq/pkg/zio/textio"
	"github.com/mccanne/zq/pkg/zio/zeekio"
	"github.com/mccanne/zq/pkg/zio/zjsonio"
	"github.com/mccanne/zq/pkg/zio/zqio"
	"github.com/mccanne/zq/pkg/zq"
	"github.com/mccanne/zq/pkg/zq/resolver"
)

type Writer struct {
	zq.WriteFlusher
	io.Closer
}

func NewWriter(writer zq.WriteFlusher, closer io.Closer) *Writer {
	return &Writer{
		WriteFlusher: writer,
		Closer:       closer,
	}
}

func (w *Writer) Close() error {
	err := w.Flush()
	cerr := w.Closer.Close()
	if err == nil {
		err = cerr
	}
	return err
}

func LookupWriter(format string, w io.WriteCloser, tc *textio.Config) *Writer {
	var f zq.WriteFlusher
	switch format {
	default:
		return nil
	case "zq":
		f = zq.NopFlusher(zqio.NewWriter(w))
	case "bzq":
		f = zq.NopFlusher(bzqio.NewWriter(w))
	case "zeek":
		f = zq.NopFlusher(zeekio.NewWriter(w))
	case "ndjson":
		f = zq.NopFlusher(ndjsonio.NewWriter(w))
		//XXX maybe zjson should be json?
	case "zjson":
		f = zq.NopFlusher(zjsonio.NewWriter(w))
	case "text":
		f = zq.NopFlusher(textio.NewWriter(w, tc))
	case "table":
		f = tableio.NewWriter(w)
	}
	return &Writer{
		WriteFlusher: f,
		Closer:       w,
	}
}

func LookupReader(format string, r io.Reader, table *resolver.Table) zq.Reader {
	switch format {
	case "zq", "zeek":
		return zqio.NewReader(r, table)
	case "ndjson":
		return ndjsonio.NewReader(r, table)
	case "zjson":
		return zjsonio.NewReader(r, table)
	case "bzq":
		return bzqio.NewReader(r, table)
	}
	return nil
}

func Extension(format string) string {
	switch format {
	case "zq":
		return ".zq"
	case "zeek":
		return ".log"
	case "ndjson":
		return ".ndjson"
	case "zjson":
		return ".ndjson"
	case "text":
		return ".txt"
	case "table":
		return ".tbl"
	case "bzq":
		return ".bzq"
	default:
		return ""
	}
}

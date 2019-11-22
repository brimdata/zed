package zsio

import (
	"io"

	"github.com/mccanne/zq/pkg/zsio/json"
	"github.com/mccanne/zq/pkg/zsio/ndjson"
	"github.com/mccanne/zq/pkg/zsio/raw"
	"github.com/mccanne/zq/pkg/zsio/table"
	"github.com/mccanne/zq/pkg/zsio/zeek"
	"github.com/mccanne/zq/pkg/zsio/zjson"
	zsonio "github.com/mccanne/zq/pkg/zsio/zson"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
)

type Writer struct {
	zson.WriteFlusher
	io.Closer
}

func NewWriter(writer zson.WriteFlusher, closer io.Closer) *Writer {
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

func LookupWriter(format string, w io.WriteCloser) *Writer {
	var f zson.WriteFlusher
	switch format {
	default:
		return nil
	case "zson":
		f = zson.NopFlusher(zsonio.NewWriter(w))
	case "zeek":
		f = zson.NopFlusher(zeek.NewWriter(w))
	case "ndjson":
		f = zson.NopFlusher(ndjson.NewWriter(w))
	case "json":
		f = json.NewWriter(w)
	case "zjson":
		f = &flusher{zjson.NewWriter(w)}
	// XXX not yet
	//case "text":
	//	return text.NewWriter(f, c.showTypes, c.showFields, c.epochDates)
	case "table":
		f = table.NewWriter(w)
	case "raw":
		f = zson.NopFlusher(raw.NewWriter(w))
	}
	return &Writer{
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
		/* XXX not yet
		case "json":
			return json.NewReader(r, table)
				case "text":
					return text.NewReader(f, c.showTypes, c.showFields, c.epochDates)

			case "table":
				return table.NewReader(r, table) */
	case "raw":
		return raw.NewReader(r, table)
	}
	return nil
}

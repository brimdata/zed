package zsio

import (
	"io"

	"github.com/mccanne/zq/pkg/zsio/json"
	"github.com/mccanne/zq/pkg/zsio/ndjson"
	"github.com/mccanne/zq/pkg/zsio/raw"
	"github.com/mccanne/zq/pkg/zsio/table"
	"github.com/mccanne/zq/pkg/zsio/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
)

type flusher struct {
	zson.Writer
}

func (f *flusher) Flush() error {
	return nil
}

func LookupWriter(format string, w io.Writer) zson.WriteFlusher {
	switch format {
	case "zson":
		return &flusher{Writer: NewWriter(w)}
	case "zeek":
		return &flusher{zeek.NewWriter(w)}
	case "ndjson":
		return &flusher{ndjson.NewWriter(w)}
	case "json":
		return json.NewWriter(w)
	//case "text":
	//	return &flusher{text.NewWriter(w, c.showTypes, c.showFields, c.epochDates)}
	case "table":
		return table.NewWriter(w)
	case "raw":
		return &flusher{raw.NewWriter(w)}
	}
	return nil
}

func LookupReader(format string, r io.Reader, table *resolver.Table) zson.Reader {
	switch format {
	case "zson", "zeek":
		return NewReader(r, table)
	case "ndjson":
		return ndjson.NewReader(r, table)
		/* XXX not yet
		case "json":
			return json.NewReader(r, table)
				case "text":
					return text.NewReader(f, c.showTypes, c.showFields, c.epochDates)

			case "table":
				return table.NewReader(r, table)
			case "raw":
				return raw.NewReader(r, table) */
	}
	return nil
}

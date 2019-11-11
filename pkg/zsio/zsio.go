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

func LookupWriter(format string, w io.WriteCloser) zson.WriteCloser {
	switch format {
	case "zson":
		return NewWriter(w)
	case "zeek":
		return zeek.NewWriter(w)
	case "ndjson":
		return ndjson.NewWriter(w)
	case "json":
		return json.NewWriter(w)
		/* XXX not yet
		case "text":
			return text.NewWriter(f, c.showTypes, c.showFields, c.epochDates)
		*/
	case "table":
		return table.NewWriter(w)
	case "raw":
		return raw.NewWriter(w)
	}
	return nil
}

func LookupReader(format string, r io.Reader, table *resolver.Table) zson.Reader {
	switch format {
	case "zson":
		return NewReader(r, table)
	case "zeek":
		return zeek.NewReader(r, table)
		/* XXX not yet
		case "ndjson":
			return ndjson.NewReader(r, table)
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

package ndjson

import (
	"bufio"
	"io"

	"github.com/mccanne/zq/pkg/zeek"
	zjson "github.com/mccanne/zq/pkg/zsio/json"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
)

type Reader struct {
	scanner  *bufio.Scanner
	resolver *resolver.Table
}

func NewReader(reader io.Reader, r *resolver.Table) *Reader {
	return &Reader{
		scanner:  bufio.NewScanner(reader),
		resolver: r,
	}
}

func (r *Reader) Read() (*zson.Record, error) {
again:
	if !r.scanner.Scan() {
		return nil, r.scanner.Err()
	}
	line := r.scanner.Bytes()
	if len(line) == 0 {
		goto again
	}
	raw, typ, err := zjson.NewRawAndType(line)
	if err != nil {
		return nil, err
	}
	desc := r.resolver.GetByColumns(typ.(*zeek.TypeRecord).Columns)
	return zson.NewRecord(desc, 0, raw), nil
}

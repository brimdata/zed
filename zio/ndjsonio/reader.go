package ndjsonio

import (
	"bytes"
	"fmt"
	"io"

	"github.com/brimsec/zq/pkg/skim"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

const (
	ReadSize    = 64 * 1024
	MaxLineSize = 50 * 1024 * 1024
)

type Reader struct {
	scanner *skim.Scanner
	parser  *Parser
	zctx    *resolver.Context
}

func NewReader(reader io.Reader, zctx *resolver.Context) *Reader {
	buffer := make([]byte, ReadSize)
	return &Reader{
		scanner: skim.NewScanner(reader, buffer, MaxLineSize),
		parser:  NewParser(zctx),
		zctx:    zctx,
	}
}

func (r *Reader) Read() (*zng.Record, error) {
again:
	line, err := r.scanner.ScanLine()
	if line == nil {
		return nil, err
	}
	line = bytes.TrimSpace(line)
	// skip empty lines
	if len(line) == 0 {
		goto again
	}
	raw, typ, err := r.parser.Parse(line)
	if err != nil {
		return nil, fmt.Errorf("line %d: %w", r.scanner.Stats.Lines, err)
	}
	outType := r.zctx.LookupTypeRecord(typ.(*zng.TypeRecord).Columns)
	return zng.NewRecordCheck(outType, 0, raw)
}

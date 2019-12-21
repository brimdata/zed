package ndjsonio

import (
	"bytes"
	"fmt"
	"io"

	"github.com/mccanne/zq/pkg/skim"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zng"
	"github.com/mccanne/zq/pkg/zng/resolver"
)

const (
	ReadSize    = 64 * 1024
	MaxLineSize = 50 * 1024 * 1024
)

type Reader struct {
	scanner  *skim.Scanner
	parser   *Parser
	resolver *resolver.Table
}

func NewReader(reader io.Reader, r *resolver.Table) *Reader {
	buffer := make([]byte, ReadSize)
	return &Reader{
		scanner:  skim.NewScanner(reader, buffer, MaxLineSize),
		parser:   NewParser(),
		resolver: r,
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
		// XXX we should be incrementing a stats counter of skipped lines.
		if err == ErrMultiTypedVector {
			goto again
		}
		return nil, fmt.Errorf("line %d: %w", r.scanner.Stats.Lines, err)
	}
	desc := r.resolver.GetByColumns(typ.(*zeek.TypeRecord).Columns)
	return zng.NewRecordCheck(desc, 0, raw)
}

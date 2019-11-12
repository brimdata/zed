package zeek

import (
	"bufio"
	"fmt"
	"io"

	"github.com/mccanne/zq/pkg/skim"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
)

const (
	ReadSize    = 64 * 1024
	MaxLineSize = 50 * 1024 * 1024
)

func scanErr(err error, n int) error {
	if err == bufio.ErrTooLong {
		return fmt.Errorf("max line size exceeded at line %d", n)
	}
	return fmt.Errorf("error encountered after %d lines: %s", n, err)
}

type ReadStats struct {
	RecordsRead int `json:"records_read"`
	BytesRead   int `json:"bytes_read"`
	BadFormat   int `json:"bad_format"`
	BadMetaData int `json:"bad_meta_data"`
	ReadFailure int `json:"read_failure"`
	LineTooLong int `json:"line_too_long"`
}

type Reader struct {
	scanner  *skim.Scanner
	parser   *parser
	stats    ReadStats
}

func NewReader(reader io.Reader, r *resolver.Table) *Reader {
	buffer := make([]byte, ReadSize)
	return &Reader{
		scanner:  skim.NewScanner(reader, buffer, MaxLineSize),
		parser:   newParser(r),
	}
}

// getline returns the next line skipping blank lines and too-long lines
// XXX for zq, we should probabluy return line-too-long error
func (r *Reader) getline() ([]byte, error) {
	for {
		line, err := r.scanner.Scan()
		if err == nil {
			if line == nil {
				return nil, nil
			}
			if len(line) <= 1 {
				// blank line, keep going
				continue
			}
			return line, nil
		}
		if err == io.EOF {
			return nil, nil
		}
		if err == skim.ErrLineTooLong {
			r.stats.LineTooLong++
			_, err = r.scanner.Skip()
			if err == nil {
				continue
			}
		}
		return nil, scanErr(err, r.stats.RecordsRead)
	}
}

func (r *Reader) Read() (*zson.Record, error) {
again:
	line, err := r.getline()
	if line == nil {
		return nil, err
	}
	// remove newline
	line = line[:len(line)-1]
	if line[0] == '#' {
		err = r.parser.parseDirective(line)
		if err != nil {
			return nil, err
		}
		goto again
	}
	rec, err := r.parser.parseValue(line)
	if err != nil {
		return nil, err
	}
	r.stats.BytesRead += len(line)
	r.stats.RecordsRead++
	return rec, nil
}

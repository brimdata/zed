package zsio

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/skim"
	zeektype "github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zsio/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/pkg/zson/resolver"
)

var (
	ErrBadFormat        = errors.New("bad format") //XXX
	ErrDescriptorExists = errors.New("descriptor already exists")
	ErrBadValue         = errors.New("bad value") //XXX
	ErrInvalidDesc      = errors.New("invalid descriptor")
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
	scanner     *skim.Scanner
	zeek        *zeek.Parser
	stats       ReadStats
	descriptors map[int]*zson.Descriptor
	resolver    *resolver.Table
	legacyVal   bool
}

func NewReader(reader io.Reader, r *resolver.Table) *Reader {
	buffer := make([]byte, ReadSize)
	return &Reader{
		scanner:     skim.NewScanner(reader, buffer, MaxLineSize),
		zeek:        zeek.NewParser(r),
		descriptors: make(map[int]*zson.Descriptor),
		resolver:    r,
	}
}

// getline returns the next line skipping blank lines and too-long lines
// XXX for zq, we should probably return line-too-long error
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
		err = r.parseDirective(line)
		if err != nil {
			return nil, err
		}
		goto again
	}
	rec, err := r.parseValue(line)
	if err != nil {
		return nil, err
	}
	r.stats.BytesRead += len(line)
	r.stats.RecordsRead++
	return rec, nil
}

func parseLeadingInt(line []byte) (val int, rest []byte, err error) {
	i := bytes.IndexByte(line, byte(':'))
	if i < 0 {
		return -1, nil, ErrBadFormat
	}
	v, err := strconv.ParseUint(string(line[:i]), 10, 32)
	if err != nil {
		return -1, nil, err
	}
	return int(v), line[i+1:], nil
}

func (r *Reader) parseDescriptor(line []byte) error {
	// #int:type
	id, rest, err := parseLeadingInt(line)
	if err != nil {
		return err
	}
	_, ok := r.descriptors[id]
	if ok {
		//XXX this should be ok... need mapper
		return ErrDescriptorExists
	}
	// XXX doesn't handle nested descriptors such as
	// #1:record[foo:int]
	// #2:record[foos:vector[1]]
	typ, err := zeektype.LookupType(string(rest))
	if err != nil {
		return err
	}

	recordType, ok := typ.(*zeektype.TypeRecord)
	if !ok {
		return ErrBadValue // XXX?
	}
	//XXX TBD we need to use a mapper that is shared by all the readers
	r.descriptors[id] = r.resolver.GetByValue(recordType)
	return nil
}

func (r *Reader) parseDirective(line []byte) error {
	if len(line) == 0 {
		return ErrBadFormat
	}
	// skip '#'
	line = line[1:]
	if len(line) == 0 {
		return ErrBadFormat
	}
	if line[0] == '!' {
		// comment
		r.legacyVal = false
		return nil
	}
	if line[0] >= '0' && line[0] <= '9' {
		r.legacyVal = false
		return r.parseDescriptor(line)
	}
	if bytes.HasPrefix(line, []byte("sort")) {
		// #sort [+-]<field>,[+-]<field>,...
		// XXX handle me
		r.legacyVal = false
		return nil
	}
	if err := r.zeek.ParseDirective(line); err != nil {
		return err
	}
	r.legacyVal = true
	return nil
}

func (r *Reader) parseValue(line []byte) (*zson.Record, error) {
	if r.legacyVal {
		return r.zeek.ParseValue(line)
	}

	// From the zson spec:
	// A regular value is encoded on a line as type descriptor
	// followed by ":" followed by a value encoding.
	id, rest, err := parseLeadingInt(line)
	if err != nil {
		return nil, err
	}

	descriptor, ok := r.descriptors[id]
	if !ok {
		return nil, ErrInvalidDesc
	}

	raw, err := zson.NewRawFromZSON(descriptor, rest)
	if err != nil {
		return nil, err
	}
	fmt.Println("READ RAW", zson.Raw(raw).String())

	record, err := zson.NewRecord(descriptor, nano.MinTs, raw), nil
	if err != nil {
		return nil, err
	}
	ts, err := record.AccessTime("ts")
	if err == nil {
		record.Ts = ts
	}
	// Ignore errors, it just means the point doesn't have a ts field
	return record, nil
}

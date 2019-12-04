package zson

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

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
	*skim.Stats
	RecordsRead int `json:"records_read"`
	BadFormat   int `json:"bad_format"`
	BadMetaData int `json:"bad_meta_data"`
	ReadFailure int `json:"read_failure"`
}

type Reader struct {
	scanner   *skim.Scanner
	zeek      *zeek.Parser
	stats     ReadStats
	mapper    *resolver.Mapper
	legacyVal bool
	ctrl      bool
	parser    *zson.Parser
}

func NewReader(reader io.Reader, r *resolver.Table) *Reader {
	buffer := make([]byte, ReadSize)
	scanner := skim.NewScanner(reader, buffer, MaxLineSize)
	return &Reader{
		scanner: scanner,
		stats:   ReadStats{Stats: &scanner.Stats},
		zeek:    zeek.NewParser(r),
		mapper:  resolver.NewMapper(r),
		parser:  zson.NewParser(),
	}
}

func NewControlReader(reader io.Reader, t *resolver.Table) *Reader {
	r := NewReader(reader, t)
	r.ctrl = true
	return r
}

func (r *Reader) Read() (*zson.Record, error) {
again:
	line, err := r.scanner.ScanLine()
	if line == nil {
		if err != nil {
			err = scanErr(err, r.stats.Lines)
		}
		return nil, err
	}
	// remove newline
	line = line[:len(line)-1]
	if line[0] == '#' {
		b, err := r.parseDirective(line)
		if err != nil {
			return nil, err
		}
		if b != nil && r.ctrl {
			return zson.NewControlRecord(b), nil
		}
		goto again
	}
	rec, err := r.parseValue(line)
	if err != nil {
		return nil, err
	}
	r.stats.RecordsRead++
	return rec, nil
}

func parseLeadingInt(line []byte) (int, uint16, []byte, error) {
	i := bytes.IndexByte(line, byte(':'))
	if i < 0 {
		return -1, 0, nil, ErrBadFormat
	}
	s := string(line[:i])
	k := strings.IndexByte(s, '.')
	var ch uint16
	if k >= 0 {
		v, err := strconv.ParseUint(s[k+1:], 10, 32)
		if err != nil {
			return -1, 0, nil, err
		}
		if v > 0xffff {
			return -1, 0, nil, fmt.Errorf("channel out of range: %d", v)
		}
		s = s[:k]
		ch = uint16(v)
	}
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return -1, 0, nil, err
	}
	return int(v), ch, line[i+1:], nil
}

func (r *Reader) parseDescriptor(line []byte) error {
	// #int:type
	id, _, rest, err := parseLeadingInt(line)
	if err != nil {
		return err
	}
	if r.mapper.Map(id) != nil {
		//XXX this should be ok... decide on this and update spec
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
	if r.mapper.Enter(id, recordType) == nil {
		// XXX this shouldn't happen
		return ErrBadValue
	}
	return nil
}

func (r *Reader) parseDirective(line []byte) ([]byte, error) {
	if len(line) == 0 {
		return nil, ErrBadFormat
	}
	// skip '#'
	line = line[1:]
	if len(line) == 0 {
		return nil, ErrBadFormat
	}
	if line[0] == '!' {
		// comment
		r.legacyVal = false
		return line[1:], nil
	}
	if line[0] >= '0' && line[0] <= '9' {
		r.legacyVal = false
		return nil, r.parseDescriptor(line)
	}
	if bytes.HasPrefix(line, []byte("sort")) {
		// #sort [+-]<field>,[+-]<field>,...
		// XXX handle me
		r.legacyVal = false
		return nil, nil
	}
	if err := r.zeek.ParseDirective(line); err != nil {
		return nil, err
	}
	r.legacyVal = true
	return nil, nil
}

func (r *Reader) parseValue(line []byte) (*zson.Record, error) {
	if r.legacyVal {
		return r.zeek.ParseValue(line)
	}

	// From the zson spec:
	// A regular value is encoded on a line as type descriptor
	// followed by ":" followed by a value encoding.
	id, ch, rest, err := parseLeadingInt(line)
	if err != nil {
		return nil, err
	}

	descriptor := r.mapper.Map(id)
	if descriptor == nil {
		return nil, ErrInvalidDesc
	}

	raw, err := r.parser.Parse(descriptor, rest)
	if err != nil {
		return nil, err
	}

	record, err := zson.NewRecordCheck(descriptor, nano.MinTs, raw)
	if err != nil {
		return nil, err
	}
	ts, err := record.AccessTime("ts")
	if err == nil {
		record.Ts = ts
	}
	record.Channel = uint16(ch)
	// Ignore errors, it just means the point doesn't have a ts field
	return record, nil
}

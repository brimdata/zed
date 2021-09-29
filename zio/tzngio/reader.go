package tzngio

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/skim"
	"github.com/brimdata/zed/zson"
)

var (
	ErrBadFormat   = errors.New("bad format") //XXX
	ErrBadValue    = errors.New("bad value")  //XXX
	ErrInvalidDesc = errors.New("invalid descriptor")
)

const (
	ReadSize    = 64 * 1024
	MaxLineSize = 50 * 1024 * 1024
)

func scanErr(err error) error {
	if err == bufio.ErrTooLong {
		return fmt.Errorf("max line size exceeded")
	}
	return err
}

type ReadStats struct {
	*skim.Stats
	RecordsRead int `json:"records_read"`
	BadFormat   int `json:"bad_format"`
	BadMetadata int `json:"bad_metadata"`
	ReadFailure int `json:"read_failure"`
	Unknown     int `json:"unknown"`
}

type Reader struct {
	scanner *skim.Scanner
	stats   ReadStats
	zctx    *zson.Context
	mapper  map[string]zed.Type
	parser  *Parser
	types   *TypeParser
}

func NewReader(reader io.Reader, zctx *zson.Context) *Reader {
	buffer := make([]byte, ReadSize)
	scanner := skim.NewScanner(reader, buffer, MaxLineSize)
	return &Reader{
		scanner: scanner,
		stats:   ReadStats{Stats: &scanner.Stats},
		zctx:    zctx,
		mapper:  make(map[string]zed.Type),
		parser:  NewParser(),
		types:   NewTypeParser(zctx),
	}
}

func (r *Reader) Read() (*zed.Record, error) {
	for {
		rec, b, err := r.ReadPayload()
		if b != nil {
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", r.stats.Lines, err)
			}
			continue
		}
		if err != nil {
			err = fmt.Errorf("line %d: %w", r.stats.Lines, err)
		}
		return rec, err
	}
}

func (r *Reader) ReadPayload() (*zed.Record, []byte, error) {
again:
	line, err := r.scanner.ScanLine()
	if line == nil {
		if err != nil {
			err = scanErr(err)
		}
		return nil, nil, err
	}
	// remove newline
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return nil, nil, ErrBadFormat
	}
	if line[0] == '#' {
		b, err := r.parseDirective(line)
		if err != nil {
			return nil, nil, err
		}
		if b != nil {
			return nil, b, nil
		}
		goto again
	}
	rec, err := r.parseValue(line)
	if err != nil {
		return nil, nil, err
	}
	r.stats.RecordsRead++
	return rec, nil, nil
}

func parseIntType(line []byte) (string, []byte, bool) {
	i := bytes.IndexByte(line, ':')
	if i <= 0 {
		return "", line, false
	}
	name := string(line[:i])
	if _, err := strconv.ParseUint(name, 10, 64); err != nil {
		return "", line, false
	}
	return name, line[i+1:], true
}

func parseAliasType(line []byte) (string, []byte, bool) {
	i := bytes.IndexByte(line, '=')
	if i <= 0 {
		return "", line, false
	}
	name := string(line[:i])
	if !zed.IsIdentifier(name) {
		return "", line, false
	}
	return name, line[i+1:], true
}

func (r *Reader) parseTypeDef(line []byte) (bool, error) {
	// #int:type (skipped past #)
	var isAlias bool
	name, rest, ok := parseIntType(line)
	if !ok {
		name, rest, ok = parseAliasType(line)
		if !ok {
			return false, ErrBadFormat
		}
		isAlias = true
	}
	typ, err := r.types.Parse(string(rest))
	if err != nil {
		return false, err
	}
	if isAlias {
		if _, ok := r.mapper[name]; ok {
			return false, errors.New("alias exists with different type")
		}
		typ, err = r.zctx.LookupTypeAlias(name, typ)
		if err != nil {
			return false, err
		}
	}
	r.mapper[name] = typ
	return true, nil
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
		return line[1:], nil
	}
	if ok, err := r.parseTypeDef(line); ok || err != nil {
		return nil, err
	}
	// XXX return an error?
	r.stats.Unknown++
	return nil, nil
}

func (r *Reader) parseType(line []byte) (zed.Type, []byte, error) {
	i := bytes.IndexByte(line, ':')
	if i <= 0 {
		return nil, nil, ErrBadFormat
	}
	id := string(line[:i])
	typ, ok := r.mapper[id]
	if !ok {
		return nil, nil, ErrInvalidDesc
	}
	return typ, line[i+1:], nil
}

func (r *Reader) parseValue(line []byte) (*zed.Record, error) {
	// From the zng spec:
	// A regular value is encoded on a line as type descriptor
	// followed by ":" followed by a value encoding.
	typ, rest, err := r.parseType(line)
	if err != nil {
		return nil, err
	}
	recType, ok := zed.AliasOf(typ).(*zed.TypeRecord)
	if !ok {
		return nil, errors.New("outer type is not a record type")
	}
	bytes, err := r.parser.Parse(recType, rest)
	if err != nil {
		return nil, err
	}
	return zed.NewRecordCheck(typ, bytes)
}

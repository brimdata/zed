// Package ndjsonio parses ndjson records. It can do basic
// transcription of json types into the corresponding zng types, or
// more advanced mapping into zng types using definitions in a
// TypeConfig.
package ndjsonio

import (
	"bytes"
	"fmt"
	"io"

	"github.com/brimsec/zq/pkg/skim"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zio/zjsonio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/buger/jsonparser"
)

const (
	ReadSize    = 64 * 1024
	MaxLineSize = 50 * 1024 * 1024
)

type ReadStats struct {
	*skim.Stats
	*typeStats
}

type Reader struct {
	scanner *skim.Scanner
	inf     inferParser
	typ     *typeParser
	zctx    *resolver.Context
	stats   ReadStats
}

func NewReader(reader io.Reader, zctx *resolver.Context) (*Reader, error) {
	_, err := zctx.LookupTypeAlias("zenum", zng.TypeString)
	if err != nil {
		return nil, err
	}

	buffer := make([]byte, ReadSize)
	scanner := skim.NewScanner(reader, buffer, MaxLineSize)
	return &Reader{
		scanner: scanner,
		stats:   ReadStats{Stats: &scanner.Stats, typeStats: &typeStats{}},
		inf:     inferParser{zctx},
		zctx:    zctx,
	}, nil
}

// A Rule contains one or more matches and the name of a descriptor
// key (in the companion Descriptors map).
type Rule struct {
	Match      map[string]string `json:"match"`
	Descriptor string            `json:"descriptor"`
}

// A TypeConfig contains a map of Descriptors, keyed by name, and a
// list of rules defining which records should be mapped into which
// descriptor.
type TypeConfig struct {
	Descriptors   map[string][]interface{} `json:"descriptors"`
	MatchingRules []Rule                   `json:"matching_rules"`
}

// typeRules is used internally and is derived from TypeConfig by
// converting its descriptors into *zng.TypeRecord s for use by the
// ndjson typed parser.
type typeRules struct {
	descriptors map[string]*zng.TypeRecord
	rules       []Rule
}

// SetTypeConfig adds a TypeConfig to the reader. Its use is optional,
// but if used, it should be called before records are processed.  In
// the absence of a TypeConfig, records are all parsed with the
// inferParser. If a TypeConfig is present, records are first parsed
// with the typeParser, and if that fails, with the inferParser.
func (r *Reader) SetTypeConfig(tc TypeConfig) error {
	tr := typeRules{
		descriptors: make(map[string]*zng.TypeRecord),
		rules:       tc.MatchingRules,
	}

	for key, columns := range tc.Descriptors {
		typeName, err := zjsonio.DecodeType(columns)
		if err != nil {
			return fmt.Errorf("error decoding type \"%s\": %s", typeName, err)
		}
		typ, err := r.zctx.LookupByName(typeName)
		if err != nil {
			return err
		}
		recType, ok := typ.(*zng.TypeRecord)
		if !ok {
			return fmt.Errorf("type not a record: \"%s\"", typeName)
		}
		tr.descriptors[key] = recType
	}
	r.typ = &typeParser{zctx: r.zctx, tr: tr, stats: r.stats.typeStats}
	return nil
}

// Parse returns a zng.Encoding slice as well as an inferred zng.Type
// from the provided JSON input. The function expects the input json to be an
// object, otherwise an error is returned.
func (r *Reader) Parse(b []byte) (zcode.Bytes, zng.Type, error) {
	val, typ, _, err := jsonparser.Get(b)
	if err != nil {
		return nil, nil, err
	}
	if typ != jsonparser.Object {
		return nil, nil, fmt.Errorf("expected JSON type to be Object but got %s", typ)
	}
	if r.typ != nil {
		zv, err := r.typ.parseObject(val)
		if err != nil {
			return nil, nil, err
		}
		return zv.Bytes, zv.Type, nil
	}

	zv, err := r.inf.parseObject(val)
	if err != nil {
		return nil, nil, err
	}
	return zv.Bytes, zv.Type, nil
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
	raw, typ, err := r.Parse(line)
	if err != nil {
		return nil, fmt.Errorf("line %d: %w", r.scanner.Stats.Lines, err)
	}
	outType := r.zctx.LookupTypeRecord(typ.(*zng.TypeRecord).Columns)
	return zng.NewRecordCheck(outType, 0, raw)
}

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

// typeRules is used internally and is derived from TypeConfig by
// converting its descriptors into *zng.TypeRecord s for use by the
// ndjson typed parser.
type typeRules struct {
	descriptors map[string]*zng.TypeRecord
	rules       []Rule
}

// ConfigureTypes adds a TypeConfig to the reader. Its should be
// called before input lines are processed. If a non-empty defaultPath
// is passed, it is used for json objects without a _path.
// In the absence of a TypeConfig, records are all parsed with the
// inferParser. If a TypeConfig is present, records are parsed
// with the typeParser.
func (r *Reader) ConfigureTypes(tc TypeConfig, defaultPath string) error {
	tr := typeRules{
		descriptors: make(map[string]*zng.TypeRecord),
		rules:       tc.Rules,
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
	r.typ = &typeParser{
		zctx:          r.zctx,
		tr:            tr,
		stats:         r.stats.typeStats,
		typeInfoCache: make(map[int]*typeInfo),
		defaultPath:   defaultPath,
	}
	return nil
}

// Parse returns a zng.Value from the provided JSON input. The
// function expects the input json to be an object, otherwise an error
// is returned.
func (r *Reader) Parse(b []byte) (zng.Value, error) {
	val, typ, _, err := jsonparser.Get(b)
	if err != nil {
		return zng.Value{}, err
	}
	if typ != jsonparser.Object {
		return zng.Value{}, fmt.Errorf("expected JSON type to be Object but got %s", typ)
	}
	if r.typ != nil {
		return r.typ.parseObject(val)
	}
	return r.inf.parseObject(val)
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
	zv, err := r.Parse(line)
	if err != nil {
		return nil, fmt.Errorf("line %d: %w", r.scanner.Stats.Lines, err)
	}
	outType := r.zctx.LookupTypeRecord(zv.Type.(*zng.TypeRecord).Columns)
	return zng.NewRecordCheck(outType, 0, zv.Bytes)
}

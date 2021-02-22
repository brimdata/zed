// Package ndjsonio parses ndjson records. It can do basic
// transcription of json types into the corresponding zng types, or
// more advanced mapping into zng types using definitions in a
// TypeConfig.
package ndjsonio

import (
	"bytes"
	"fmt"
	"io"
	"regexp"

	"github.com/brimsec/zq/pkg/skim"
	"github.com/brimsec/zq/zio/ndjsonio/compat"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/buger/jsonparser"
)

const (
	ReadSize    = 64 * 1024
	MaxLineSize = 50 * 1024 * 1024
)

// x509.14:00:00-15:00:00.log.gz (open source zeek)
// x509_20191101_14:00:00-15:00:00+0000.log.gz (corelight)
const DefaultPathRegexp = `([a-zA-Z0-9_]+)(?:\.|_\d{8}_)\d\d:\d\d:\d\d\-\d\d:\d\d:\d\d(?:[+\-]\d{4})?\.log(?:$|\.gz)`

type ReaderOpts struct {
	TypeConfig *TypeConfig
	PathRegexp string
	Warnings   chan<- string
}

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
	types   *tzngio.TypeParser
}

func NewReader(reader io.Reader, zctx *resolver.Context, opts ReaderOpts, filepath string) (*Reader, error) {
	// Note: we add hardwired aliases for "port" to "uint16" when reading
	// *any* json file but they are only used when the schema mapper
	// (aka typings config) references such types from a configured schema.
	// However, the schema mapper should be responsible for creating these
	// aliases according to its configuration.  See issue #1427.
	_, err := zctx.LookupTypeAlias("zenum", zng.TypeString)
	if err != nil {
		return nil, err
	}
	_, err = zctx.LookupTypeAlias("port", zng.TypeUint16)
	if err != nil {
		return nil, err
	}
	buffer := make([]byte, ReadSize)
	scanner := skim.NewScanner(reader, buffer, MaxLineSize)
	r := &Reader{
		scanner: scanner,
		stats:   ReadStats{Stats: &scanner.Stats, typeStats: &typeStats{}},
		inf:     inferParser{zctx},
		zctx:    zctx,
		types:   tzngio.NewTypeParser(zctx),
	}
	if opts.TypeConfig != nil {
		var path string
		re, err := regexp.Compile(opts.PathRegexp)
		if err != nil {
			return nil, err
		}
		//XXX why do we do this this way?
		match := re.FindStringSubmatch(filepath)
		if len(match) == 2 {
			path = match[1]
		}
		if err = r.configureTypes(*opts.TypeConfig, path, opts.Warnings); err != nil {
			return nil, err
		}
	}
	return r, nil
}

// typeRules is used internally and is derived from TypeConfig by
// converting its descriptors into *zng.TypeRecord s for use by the
// ndjson typed parser.
type typeRules struct {
	descriptors map[string]*zng.TypeRecord
	rules       []Rule
}

// configureTypes adds a TypeConfig to the reader. Its should be
// called before input lines are processed. If a non-empty defaultPath
// is passed, it is used for json objects without a _path.
// In the absence of a TypeConfig, records are all parsed with the
// inferParser. If a TypeConfig is present, records are parsed
// with the typeParser.
func (r *Reader) configureTypes(tc TypeConfig, defaultPath string, warn chan<- string) error {
	tr := typeRules{
		descriptors: make(map[string]*zng.TypeRecord),
		rules:       tc.Rules,
	}

	for key, columns := range tc.Descriptors {
		typeName, err := compat.DecodeType(columns)
		if err != nil {
			return fmt.Errorf("error decoding type \"%s\": %s", typeName, err)
		}
		typ, err := r.types.Parse(typeName)
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
		passUnknowns:  tc.PassUnknowns,
		warn:          warn,
		warnSent:      make(map[string]struct{}),
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
		return r.typ.parseObject(val, r.inf)
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
	outType, err := r.zctx.LookupTypeRecord(zv.Type.(*zng.TypeRecord).Columns)
	if err != nil {
		return nil, err
	}
	return zng.NewRecordCheck(outType, zv.Bytes)
}

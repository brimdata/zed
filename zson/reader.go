package zson

import (
	"fmt"
	"io"

	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type Reader struct {
	reader   io.Reader
	zctx     *resolver.Context
	parser   *Parser
	analyzer Analyzer
	builder  *Builder
}

func NewReader(r io.Reader, zctx *resolver.Context) *Reader {
	return &Reader{
		reader:   r,
		zctx:     zctx,
		analyzer: NewAnalyzer(),
		builder:  NewBuilder(),
	}
}

func (r *Reader) Read() (*zng.Record, error) {
	if r.parser == nil {
		var err error
		r.parser, err = NewParser(r.reader)
		if err != nil {
			return nil, err
		}
	}
	ast, err := r.parser.ParseValue()
	if ast == nil || err != nil {
		return nil, err
	}
	val, err := r.analyzer.ConvertValue(r.zctx, ast)
	if err != nil {
		return nil, err
	}
	zv, err := r.builder.Build(val)
	if err != nil {
		return nil, err
	}
	//XXX Ideally, we'd like to set the type of the record to val.TypeOf(),
	// e.g., in case the record type's is a typedef name (i.e., alias), but
	// record types currently must be zng.TypeRecord. See issue #1801.
	if recType, ok := zng.AliasedType(zv.Type).(*zng.TypeRecord); ok {
		return zng.NewRecord(recType, zv.Bytes), nil
	}
	// ZSON can represent value streams that aren't records,
	// but we handle only top-level records here.
	return nil, fmt.Errorf("top-level ZSON value not a record: %s", zv.Type.ZSON())
}

package zson

import (
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Reader struct {
	reader   io.Reader
	zctx     *zed.Context
	parser   *Parser
	analyzer Analyzer
	builder  *zcode.Builder
}

func NewReader(r io.Reader, zctx *zed.Context) *Reader {
	return &Reader{
		reader:   r,
		zctx:     zctx,
		analyzer: NewAnalyzer(),
		builder:  zcode.NewBuilder(),
	}
}

func (r *Reader) Read() (*zed.Value, error) {
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
	zv, err := Build(r.builder, val)
	if err != nil {
		return nil, err
	}
	return zed.NewValue(zv.Type, zv.Bytes), nil
}

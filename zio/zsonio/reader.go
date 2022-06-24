package zsonio

import (
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

type Reader struct {
	reader   io.Reader
	zctx     *zed.Context
	parser   *zson.Parser
	analyzer zson.Analyzer
	builder  *zcode.Builder
}

func NewReader(zctx *zed.Context, r io.Reader) *Reader {
	return &Reader{
		reader:   r,
		zctx:     zctx,
		analyzer: zson.NewAnalyzer(),
		builder:  zcode.NewBuilder(),
	}
}

func (r *Reader) Read() (*zed.Value, error) {
	if r.parser == nil {
		r.parser = zson.NewParser(r.reader)
	}
	ast, err := r.parser.ParseValue()
	if ast == nil || err != nil {
		return nil, err
	}
	val, err := r.analyzer.ConvertValue(r.zctx, ast)
	if err != nil {
		return nil, err
	}
	zv, err := zson.Build(r.builder, val)
	if err != nil {
		return nil, err
	}
	return zed.NewValue(zv.Type, zv.Bytes), nil
}

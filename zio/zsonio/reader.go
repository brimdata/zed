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
	val      zed.Value
	arena    *zed.Arena
}

func NewReader(a *zed.Arena, r io.Reader) *Reader {
	return &Reader{
		reader:   r,
		zctx:     a.Zctx(),
		analyzer: zson.NewAnalyzer(),
		builder:  zcode.NewBuilder(),
		arena:    a,
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
	r.val, err = zson.Build(r.arena, r.builder, val)
	return &r.val, err
}

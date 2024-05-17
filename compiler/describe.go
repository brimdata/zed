package compiler

import (
	"context"
	"errors"

	"github.com/brimdata/zed/compiler/data"
	"github.com/brimdata/zed/compiler/describe"
	"github.com/brimdata/zed/compiler/semantic"
	"github.com/brimdata/zed/lakeparse"
)

func Describe(ctx context.Context, query string, src *data.Source, head *lakeparse.Commitish) (*describe.Info, error) {
	seq, err := Parse(query)
	if err != nil {
		return nil, err
	}
	if len(seq) == 0 {
		return nil, errors.New("internal error: AST seq cannot be empty")
	}
	entry, err := semantic.AnalyzeAddSource(ctx, seq, src, head)
	if err != nil {
		return nil, err
	}
	return describe.Analyze(ctx, src, entry)
}

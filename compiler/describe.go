package compiler

import (
	"context"

	"github.com/brimdata/zed/compiler/data"
	"github.com/brimdata/zed/compiler/describe"
	"github.com/brimdata/zed/compiler/parser"
	"github.com/brimdata/zed/compiler/semantic"
	"github.com/brimdata/zed/lakeparse"
)

func Describe(ctx context.Context, query string, src *data.Source, head *lakeparse.Commitish) (*describe.Info, error) {
	seq, sset, err := Parse(query)
	if err != nil {
		return nil, err
	}
	entry, err := semantic.AnalyzeAddSource(ctx, seq, src, head)
	if err != nil {
		if list, ok := err.(parser.ErrorList); ok {
			list.SetSourceSet(sset)
		}
		return nil, err
	}
	return describe.Analyze(ctx, src, entry)
}

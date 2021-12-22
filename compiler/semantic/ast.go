package semantic

import (
	"context"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/proc"
)

// Analyze analysis the AST and prepares it for runtime compilation.
func Analyze(ctx context.Context, seq *ast.Sequential, adaptor proc.DataAdaptor, head *lakeparse.Commitish) (*dag.Sequential, error) {
	entry, err := semSequential(ctx, NewScope(), seq, adaptor, head)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func isFrom(seq *ast.Sequential) bool {
	if len(seq.Procs) == 0 {
		return false
	}
	_, ok := seq.Procs[0].(*ast.From)
	return ok
}

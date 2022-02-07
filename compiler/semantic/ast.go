package semantic

import (
	"context"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/proc"
)

// Analyze performs a semantic analysis of the AST, translating it from AST
// to DAG form, resolving syntax ambiguities, and performing constant propagation.
// After semantic analysis, the DAG is ready for either optimization or compilation.
func Analyze(ctx context.Context, seq *ast.Sequential, adaptor proc.DataAdaptor, head *lakeparse.Commitish) (*dag.Sequential, error) {
	return semSequential(ctx, NewScope(), seq, adaptor, head)
}

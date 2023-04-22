package semantic

import (
	"context"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/data"
	"github.com/brimdata/zed/lakeparse"
)

// Analyze performs a semantic analysis of the AST, translating it from AST
// to DAG form, resolving syntax ambiguities, and performing constant propagation.
// After semantic analysis, the DAG is ready for either optimization or compilation.
func Analyze(ctx context.Context, scope *ast.Scope, source *data.Source, head *lakeparse.Commitish) (*dag.Scope, error) {
	return semScope(ctx, NewScope(), scope, source, head)
}

package semantic

import (
	"context"
	"errors"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/proc"
)

// Analyze analysis the AST and prepares it for runtime compilation.
func Analyze(ctx context.Context, seq *ast.Sequential, constsAST []ast.Proc, adaptor proc.DataAdaptor, head *lakeparse.Commitish) (*dag.Sequential, []dag.Op, error) {
	if !isFrom(seq) {
		return nil, nil, errors.New("Zed program does not begin with a data source")
	}
	scope := NewScope()
	scope.Enter()
	consts, err := semConsts(scope, constsAST)
	if err != nil {
		return nil, nil, err
	}
	entry, err := semSequential(ctx, scope, seq, adaptor, head)
	if err != nil {
		return nil, nil, err
	}
	return entry, consts, nil
}

func isFrom(seq *ast.Sequential) bool {
	if len(seq.Procs) == 0 {
		return false
	}
	_, ok := seq.Procs[0].(*ast.From)
	return ok
}

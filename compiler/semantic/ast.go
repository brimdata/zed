package semantic

import (
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
)

// Analyze analysis the AST and prepares it for runtime compilation.
func Analyze(p ast.Proc) (dag.Op, []dag.Op, error) {
	scope := NewScope()
	scope.Enter()
	consts, err := semConsts(nil, scope, p)
	if err != nil {
		return nil, nil, err
	}
	entry, err := semProc(scope, p)
	if err != nil {
		return nil, nil, err
	}
	return entry, consts, nil
}

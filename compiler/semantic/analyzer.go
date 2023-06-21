package semantic

import (
	"context"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/data"
	"github.com/brimdata/zed/lakeparse"
)

// Analyze performs a semantic analysis of the AST, translating it from AST
// to DAG form, resolving syntax ambiguities, and performing constant propagation.
// After semantic analysis, the DAG is ready for either optimization or compilation.
func Analyze(ctx context.Context, seq ast.Seq, source *data.Source, head *lakeparse.Commitish) (dag.Seq, error) {
	return newAnalyzer(ctx, source, head).semSeq(seq)
}

type analyzer struct {
	ctx       context.Context
	head      *lakeparse.Commitish
	opDeclMap map[*dag.UserOp]*opDecl
	opPath    []*dag.UserOp
	source    *data.Source
	scope     *Scope
	zctx      *zed.Context

	// opDecl is the current operator declaration being analyzed.
	opDecl *opDecl
}

func newAnalyzer(ctx context.Context, source *data.Source, head *lakeparse.Commitish) *analyzer {
	return &analyzer{
		ctx:       ctx,
		head:      head,
		opDeclMap: make(map[*dag.UserOp]*opDecl),
		source:    source,
		scope:     NewScope(nil),
		zctx:      zed.NewContext(),
	}
}

func (a *analyzer) enterScope() {
	a.scope = NewScope(a.scope)
}

func (a *analyzer) exitScope() {
	a.scope = a.scope.parent
}

type opDecl struct {
	ast   *ast.OpDecl
	deps  []*dag.UserOp
	scope *Scope // parent scope of op declaration.
}

func (a *analyzer) appendOpPath(op *dag.UserOp) error {
	a.opPath = append(a.opPath, op)
	for i := len(a.opPath) - 2; i >= 0; i-- {
		if a.opPath[i] == op {
			return opCycleError(a.opPath)
		}
	}
	return nil
}

type opCycleError []*dag.UserOp

func (e opCycleError) Error() string {
	b := "operator cycle found: "
	for i, op := range e {
		if i > 0 {
			b += " -> "
		}
		b += op.Name
	}
	return b
}

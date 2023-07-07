package semantic

import (
	"context"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/data"
	"github.com/brimdata/zed/compiler/kernel"
	"github.com/brimdata/zed/lakeparse"
)

// Analyze performs a semantic analysis of the AST, translating it from AST
// to DAG form, resolving syntax ambiguities, and performing constant propagation.
// After semantic analysis, the DAG is ready for either optimization or compilation.
func Analyze(ctx context.Context, seq ast.Seq, source *data.Source, head *lakeparse.Commitish) (dag.Seq, error) {
	a := newAnalyzer(ctx, source, head)
	s, err := a.semSeq(seq)
	if err != nil {
		return nil, err
	}
	op, err := a.buildFrom(s[0])
	if err != nil {
		return nil, err
	}
	if op != nil {
		s.Prepend(op)
	}
	return s, nil
}

type analyzer struct {
	ctx       context.Context
	head      *lakeparse.Commitish
	opDeclMap map[*dag.UserOp]*opDecl
	opStack   []*dag.UserOp
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

func (a *analyzer) buildFrom(op dag.Op) (dag.Op, error) {
	switch op := op.(type) {
	case *dag.FileScan, *dag.HTTPScan, *dag.PoolScan, *dag.LakeMetaScan, *dag.PoolMetaScan, *dag.CommitMetaScan, *dag.DeleteScan:
		return nil, nil
	case *dag.Fork:
		return a.buildFrom(op.Paths[0][0])
	case *dag.Scope:
		return a.buildFrom(op.Body[0])
	case *dag.UserOpCall:
		return a.buildFrom(op.Body[0])
	}
	// No from so add a source.
	if a.head == nil {
		return &kernel.Reader{}, nil
	}
	pool := &ast.Pool{
		Kind: "Pool",
		Spec: ast.PoolSpec{
			Pool: &ast.String{
				Kind: "String",
				Text: "HEAD",
			},
		},
	}
	ops, err := a.semPool(pool)
	if err != nil {
		return nil, err
	}
	return ops[0], nil
}

type opDecl struct {
	ast   *ast.OpDecl
	deps  []*dag.UserOp
	scope *Scope // parent scope of op declaration.
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

package semantic

import (
	"context"
	"fmt"

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
func Analyze(ctx context.Context, seq ast.Seq, source *data.Source, head *lakeparse.Commitish, addFilters []ast.Expr) (dag.Seq, error) {
	a := newAnalyzer(ctx, source, head)
	s, err := a.semSeq(seq)
	if err != nil {
		return nil, err
	}
	return a.addFilters(s, addFilters)
}

// AnalyzeAddSource is the same as Analyze but it adds a default source if the
// DAG does not have one.
func AnalyzeAddSource(ctx context.Context, seq ast.Seq, source *data.Source, head *lakeparse.Commitish, addFilters []ast.Expr) (dag.Seq, error) {
	a := newAnalyzer(ctx, source, head)
	s, err := a.semSeq(seq)
	if err != nil {
		return nil, err
	}
	if !HasSource(s) {
		if err = a.addDefaultSource(&s); err != nil {
			return nil, err
		}
	}
	return a.addFilters(s, addFilters)
}

type analyzer struct {
	ctx     context.Context
	head    *lakeparse.Commitish
	opStack []*ast.OpDecl
	source  *data.Source
	scope   *Scope
	zctx    *zed.Context
}

func newAnalyzer(ctx context.Context, source *data.Source, head *lakeparse.Commitish) *analyzer {
	return &analyzer{
		ctx:    ctx,
		head:   head,
		source: source,
		scope:  NewScope(nil),
		zctx:   zed.NewContext(),
	}
}

func HasSource(seq dag.Seq) bool {
	switch op := seq[0].(type) {
	case *dag.FileScan, *dag.HTTPScan, *dag.PoolScan, *dag.LakeMetaScan, *dag.PoolMetaScan, *dag.CommitMetaScan, *dag.DeleteScan, *kernel.Reader:
		return true
	case *dag.Fork:
		return HasSource(op.Paths[0])
	case *dag.Scope:
		return HasSource(op.Body)
	}
	return false
}

func (a *analyzer) addDefaultSource(seq *dag.Seq) error {
	if HasSource(*seq) {
		return nil
	}
	// No from so add a source.
	if a.head == nil {
		seq.Prepend(&kernel.Reader{})
		return nil
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
		return err
	}
	seq.Prepend(ops[0])
	return nil
}

func StartsWithYield(seq dag.Seq) bool {
	switch op := seq[0].(type) {
	case *dag.Yield:
		return true
	case *dag.Scope:
		return StartsWithYield(op.Body)
	}
	return false
}

func (a *analyzer) addFilters(seq dag.Seq, exprs []ast.Expr) (dag.Seq, error) {
	out, err := a.semExprs(exprs)
	if err != nil {
		return nil, err
	}
	var filters dag.Seq
	for i, e := range out {
		if !isBool(e) {
			return nil, fmt.Errorf("filter %d: expression does not result in a boolean value", i)
		}
		filters = append(filters, &dag.Filter{Kind: "Filter", Expr: e})
	}
	return append(dag.Seq{seq[0]}, append(filters, seq[1:]...)...), nil
}

func (a *analyzer) enterScope() {
	a.scope = NewScope(a.scope)
}

func (a *analyzer) exitScope() {
	a.scope = a.scope.parent
}

type opDecl struct {
	ast   *ast.OpDecl
	scope *Scope // parent scope of op declaration.
}

type opCycleError []*ast.OpDecl

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

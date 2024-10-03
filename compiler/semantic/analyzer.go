package semantic

import (
	"context"
	"errors"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/data"
	"github.com/brimdata/zed/compiler/parser"
	"github.com/brimdata/zed/lakeparse"
)

// Analyze performs a semantic analysis of the AST, translating it from AST
// to DAG form, resolving syntax ambiguities, and performing constant propagation.
// After semantic analysis, the DAG is ready for either optimization or compilation.
func Analyze(ctx context.Context, seq ast.Seq, source *data.Source, head *lakeparse.Commitish) (dag.Seq, error) {
	a := newAnalyzer(ctx, source, head)
	s := a.semSeq(seq)
	s, _ = a.checkSeqOutputs(true, s)
	if a.errors != nil {
		return nil, a.errors
	}
	return s, nil
}

// AnalyzeAddSource is the same as Analyze but it adds a default source if the
// DAG does not have one.
func AnalyzeAddSource(ctx context.Context, seq ast.Seq, source *data.Source, head *lakeparse.Commitish) (dag.Seq, error) {
	a := newAnalyzer(ctx, source, head)
	s := a.semSeq(seq)
	s, _ = a.checkSeqOutputs(true, s)
	if a.errors != nil {
		return nil, a.errors
	}
	if !HasSource(s) {
		if err := AddDefaultSource(ctx, &s, source, head); err != nil {
			return nil, err
		}
	}
	return s, nil
}

type analyzer struct {
	ctx     context.Context
	errors  parser.ErrorList
	head    *lakeparse.Commitish
	opStack []*ast.OpDecl
	outputs map[*dag.Output]ast.Node
	source  *data.Source
	scope   *Scope
	zctx    *zed.Context
}

func newAnalyzer(ctx context.Context, source *data.Source, head *lakeparse.Commitish) *analyzer {
	return &analyzer{
		ctx:     ctx,
		head:    head,
		outputs: make(map[*dag.Output]ast.Node),
		source:  source,
		scope:   NewScope(nil),
		zctx:    zed.NewContext(),
	}
}

func HasSource(seq dag.Seq) bool {
	switch op := seq[0].(type) {
	case *dag.FileScan, *dag.HTTPScan, *dag.PoolScan, *dag.LakeMetaScan, *dag.PoolMetaScan, *dag.CommitMetaScan, *dag.DeleteScan:
		return true
	case *dag.Fork:
		return HasSource(op.Paths[0])
	case *dag.Scope:
		return HasSource(op.Body)
	}
	return false
}

func AddDefaultSource(ctx context.Context, seq *dag.Seq, source *data.Source, head *lakeparse.Commitish) error {
	if HasSource(*seq) {
		return nil
	}
	// No from so add a source.
	if head == nil {
		seq.Prepend(&dag.DefaultScan{Kind: "DefaultScan"})
		return nil
	}
	// Verify pool exists for HEAD
	if _, err := source.PoolID(ctx, head.Pool); err != nil {
		return err
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
	ops := newAnalyzer(ctx, source, head).semPool(pool)
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

func (a *analyzer) enterScope() {
	a.scope = NewScope(a.scope)
}

func (a *analyzer) exitScope() {
	a.scope = a.scope.parent
}

type opDecl struct {
	ast   *ast.OpDecl
	scope *Scope // parent scope of op declaration.
	bad   bool
}

type opCycleError []*ast.OpDecl

func (e opCycleError) Error() string {
	b := "operator cycle found: "
	for i, op := range e {
		if i > 0 {
			b += " -> "
		}
		b += op.Name.Name
	}
	return b
}

func badExpr() dag.Expr {
	return &dag.BadExpr{Kind: "BadExpr"}
}

func badOp() dag.Op {
	return &dag.BadOp{Kind: "BadOp"}
}

func (a *analyzer) error(n ast.Node, err error) {
	pos, end := -1, -1
	if n != nil {
		pos, end = n.Pos(), n.End()
	}
	a.errors.Append(err.Error(), pos, end)
}

func (a *analyzer) checkSeqOutputs(isLeaf bool, seq dag.Seq) (dag.Seq, bool) {
	if len(seq) == 0 {
		return seq, false
	}
	// - Report an error in any outputs are not located in the leaves.
	// - Add output operators to any leaves where they do not exist.
	last := len(seq) - 1
	var blocked bool
	for i, o := range seq {
		if blocked {
			// XXX We need to map all DAG ops to their AST equivalent so we can
			// get location specific errors.
			a.error(nil, errors.New("unreachable operator"))
			return seq, true
		}
		blocked = a.checkOpOutput(o, isLeaf, last == i)
	}
	switch seq[last].(type) {
	case *dag.Scope, *dag.Output, *dag.Scatter, *dag.Fork, *dag.Switch, *dag.Mirror, *dag.Over:
	default:
		if isLeaf {
			seq.Append(&dag.Output{Kind: "Output", Name: "main"})
			blocked = true
		}
	}
	return seq, blocked
}

func (a *analyzer) checkOpOutput(o dag.Op, isLeafSeq bool, isLast bool) bool {
	var blocked bool
	switch o := o.(type) {
	case *dag.Output:
		blocked = true
	case *dag.Scope:
		o.Body, blocked = a.checkSeqOutputs(isLast && isLeafSeq, o.Body)
	case *dag.Scatter:
		blocked = true
		for k := range o.Paths {
			var pathBlocked bool
			o.Paths[k], pathBlocked = a.checkSeqOutputs(isLast && isLeafSeq, o.Paths[k])
			blocked = blocked && pathBlocked
		}
	case *dag.Over:
		o.Body, blocked = a.checkSeqOutputs(isLast && isLeafSeq, o.Body)
	case *dag.Fork:
		blocked = true
		for k := range o.Paths {
			var pathBlocked bool
			o.Paths[k], pathBlocked = a.checkSeqOutputs(isLast && isLeafSeq, o.Paths[k])
			blocked = blocked && pathBlocked
		}
	case *dag.Switch:
		blocked = true
		for k := range o.Cases {
			var pathBlocked bool
			o.Cases[k].Path, pathBlocked = a.checkSeqOutputs(isLast && isLeafSeq, o.Cases[k].Path)
			blocked = blocked && pathBlocked
		}
	case *dag.Mirror:
		var mainOut, mirrorOut bool
		o.Main, mainOut = a.checkSeqOutputs(isLast && isLeafSeq, o.Main)
		o.Mirror, mirrorOut = a.checkSeqOutputs(isLast && isLeafSeq, o.Mirror)
		blocked = mainOut && mirrorOut
	}
	return blocked
}

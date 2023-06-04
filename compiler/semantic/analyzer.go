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
	a := newAnalyzer(ctx, source, head)
	s, err := a.check(seq)
	if err != nil {
		return nil, err
	}
	if err := a.checkOpCycle(); err != nil {
		return nil, err
	}
	for _, f := range a.resolve {
		if err := f(); err != nil {
			return nil, err
		}
	}
	return s, nil
}

type analyzer struct {
	ctx       context.Context
	head      *lakeparse.Commitish
	opDeclMap map[*dag.UserOp]*opDecl
	resolve   []func() error
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

func (a *analyzer) check(seq ast.Seq) (dag.Seq, error) {
	return a.semSeq(seq)
}

func (a *analyzer) enterScope() {
	a.scope = NewScope(a.scope)
}

func (a *analyzer) exitScope() {
	a.scope = a.scope.parent
}

type opDecl struct {
	op   *dag.UserOp
	deps []*dag.UserOp
}

func (a *analyzer) checkOpCycle() error {
	visited := make(map[*dag.UserOp]bool)
	onStack := make(map[*dag.UserOp]bool)
	if p := a.checkScopeOpCycle(a.scope, visited, onStack); p != nil {
		return errOpCycle(p)
	}
	return nil
}

func (a *analyzer) checkScopeOpCycle(scope *Scope, visited, onStack map[*dag.UserOp]bool) []*dag.UserOp {
	for _, e := range scope.sortedEntries() {
		if op, ok := e.ref.(*dag.UserOp); ok {
			if p := a.isCyclic(op, visited, onStack, nil); p != nil {
				return p
			}
		}
	}
	for _, child := range scope.children {
		if p := a.checkScopeOpCycle(child, visited, onStack); p != nil {
			return p
		}
	}
	return nil
}

func (a *analyzer) isCyclic(op *dag.UserOp, visited, onStack map[*dag.UserOp]bool, path []*dag.UserOp) []*dag.UserOp {
	path = append(path, op)
	visited[op], onStack[op] = true, true
	for _, neighbor := range a.opDeclMap[op].deps {
		if !visited[neighbor] {
			if p := a.isCyclic(neighbor, visited, onStack, path); p != nil {
				return p
			}
		} else if onStack[neighbor] {
			path = append(path, neighbor)
			return path
		}
	}
	onStack[op] = false
	return nil
}

type errOpCycle []*dag.UserOp

func (e errOpCycle) Error() string {
	b := "operator cycle found: "
	for i, op := range e {
		if i > 0 {
			b += " -> "
		}
		b += op.Name
	}
	return b
}

package compiler

import (
	"github.com/brimsec/zq/compiler/ast"
	"github.com/brimsec/zq/compiler/kernel"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zng/resolver"
)

type Program struct {
	ast     *ast.Program
	dsl     *kernel.DSL
	scope   *Scope
	outputs []proc.Interface
	pctx    *proc.Context
	filter  kernel.BoolExpr
}

func NewProgram(p *ast.Program, pctx *proc.Context) *Program {
	scope := newScope()
	// Create the global scope.  We don't call exit because we want
	// it to stick around.
	scope.Enter()
	return &Program{
		ast:   p,
		pctx:  pctx,
		scope: scope,
	}
}

func (p *Program) Outputs() []proc.Interface {
	return p.outputs
}

func (p *Program) Filter() *Filter {
	return p.filter
}

func (p *Program) Compile(custom Hook, inputs []proc.Interface) error {
	if err := compileConsts(p.pctx.TypeContext, p.scope, p.ast.Consts); err != nil {
		return err
	}
	if err := compileTypes(p.pctx.TypeContext, p.scope, p.ast.Types); err != nil {
		return err
	}
	entry := p.entry
	if entry == nil {
		entry = passProc
	}
	outputs, err := compile(custom, entry, p.pctx, p.scope, inputs)
	if err != nil {
		return err
	}
	p.outputs = outputs
	return nil
}

func (p *Program) Optimize(sortKey field.Static, sortReversed bool) error {
	if p.entry == nil {
		return nil
	}
	newEntry, err := SemanticTransform(p.entry)
	if err != nil {
		return err
	}
	p.entry = newEntry
	if sortKey != nil {
		setGroupByProcInputSortDir(newEntry, sortKey, zbufDirInt(sortReversed))
	}
	filterExpr, liftedEntry := liftFilter(newEntry)
	if filterExpr != nil {
		p.filter = NewFilter(p.pctx.TypeContext, filterExpr)
		p.entry = liftedEntry
	}
	return nil
}

func (p *Program) IsParallelizable(sortKey field.Static, sortReversed bool) bool {
	return isParallelizable(p.entry, sortKey, sortReversed)
}

func (p *Program) Parallelize(nway int, sortKey field.Static, sortReversed bool) bool {
	var ok bool
	p.entry, ok = parallelize(p.entry, nway, sortKey, sortReversed)
	return ok
}

func compileConsts(zctx *resolver.Context, scope *Scope, consts []ast.Const) error {
	//TBD
	return nil
}

func compileTypes(zctx *resolver.Context, scope *Scope, types []ast.TypeConst) error {
	//TBD
	return nil
}

//XXX this should be methods on Program...

var _ zbuf.Filter = (*Filter)(nil)

func (p *Program) AsFilter() (expr.Filter, error) {
	if p.filter == nil {
		return nil, nil
	}
	// XXX nil scope... when we implement global scope, the filters
	// will need access to it.
	// XXX this will change soon to DSL
	return kernel.CompileFilter(f.zctx, nil, p.filter)
}

func (p *Program) AsBufferFilter() (*expr.BufferFilter, error) {
	if p.filter == nil {
		return nil, nil
	}
	return kernel.CompileBufferFilter(p.filter)
}

func (f *Filter) AST() ast.Expression {
	//XXX change to dsl
	return p.filter
}

func (f *Filter) AsProc() ast.Proc {
	//XXX change to dsl
	return ast.FilterToProc(f.ast)
}

package semantic

import (
	"fmt"
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/kernel"
	"github.com/brimdata/zed/zson"
)

type Scope struct {
	zctx  *zed.Context
	stack []*Binder
}

func NewScope() *Scope {
	return &Scope{zctx: zed.NewContext()}
}

func (s *Scope) tos() *Binder {
	return s.stack[len(s.stack)-1]
}

func (s *Scope) EnterOp(name string) {
	s.Enter()
	s.tos().op = name
}

func (s *Scope) Enter() {
	s.stack = append(s.stack, NewBinder())
}

func (s *Scope) Exit() {
	s.stack = s.stack[:len(s.stack)-1]
}

func (s *Scope) DefineVar(name string) error {
	b := s.tos()
	if _, ok := b.symbols[name]; ok {
		return fmt.Errorf("symbol %q redefined", name)
	}
	ref := &dag.Var{
		Kind: "Var",
		Name: name,
		Slot: s.nvars(),
	}
	b.Define(name, ref)
	b.nvar++
	return nil
}

func (s *Scope) DefineAs(name string) error {
	b := s.tos()
	if _, ok := b.symbols[name]; ok {
		return fmt.Errorf("symbol %q redefined", name)
	}
	// We add the symbol to the table but don't bump nvars because
	// it's not a var and doesn't take a slot in the batch vars.
	b.Define(name, &dag.This{Kind: "This"})
	return nil
}

func (s *Scope) DefineFunc(f *dag.Func) error {
	b := s.tos()
	if _, ok := b.symbols[f.Name]; ok {
		return fmt.Errorf("symbol %q redefined", f.Name)
	}
	b.Define(f.Name, f)
	return nil
}

func (s *Scope) DefineOp(o *ast.OpDecl) error {
	b := s.tos()
	if _, ok := b.symbols[o.Name]; ok {
		return fmt.Errorf("symbol %q redefined", o.Name)
	}
	b.Define(o.Name, o)
	return nil
}

func (s *Scope) DefineConst(name string, def dag.Expr) error {
	b := s.tos()
	if _, ok := b.symbols[name]; ok {
		return fmt.Errorf("symbol %q redefined", name)
	}
	val, err := kernel.EvalAtCompileTime(s.zctx, def)
	if err != nil {
		return err
	}
	if val.IsError() {
		if val.IsMissing() {
			return fmt.Errorf("const %q: cannot have variable dependency", name)
		} else {
			return fmt.Errorf("const %q: %q", name, string(val.Bytes))
		}
	}
	literal := &dag.Literal{
		Kind:  "Literal",
		Value: zson.MustFormatValue(val),
	}
	b.Define(name, literal)
	return nil
}

func (s *Scope) LookupExpr(name string) (dag.Expr, error) {
	for k := len(s.stack) - 1; k >= 0; k-- {
		if entry, ok := s.stack[k].symbols[name]; ok {
			e, ok := entry.ref.(dag.Expr)
			if !ok {
				return nil, fmt.Errorf("symbol %q is not an expression", name)
			}
			entry.refcnt++
			return e, nil
		}
	}
	return nil, nil
}

func (s *Scope) LookupOp(name string) (*ast.OpDecl, error) {
	calls := []string{name}
	for k := len(s.stack) - 1; k >= 0; k-- {
		b := s.stack[k]
		if b.op != "" {
			calls = append(calls, b.op)
		}
		if b.op == name {
			return nil, errOpCycle(calls)
		}
		if entry, ok := b.symbols[name]; ok {
			d, ok := entry.ref.(*ast.OpDecl)
			if !ok {
				return nil, fmt.Errorf("symbol %q is not an op", name)
			}
			entry.refcnt++
			return d, nil
		}
	}
	return nil, nil
}

type errOpCycle []string

func (e errOpCycle) Error() string {
	var b strings.Builder
	b.WriteString("op cycle found: ")
	for k := len(e) - 1; k >= 0; k-- {
		b.WriteString(e[k])
		if k > 0 {
			b.WriteString(" -> ")
		}
	}
	return b.String()
}

func (s *Scope) nvars() int {
	var n int
	for _, scope := range s.stack {
		n += scope.nvar
	}
	return n
}

type entry struct {
	ref    any
	refcnt int
}

type Binder struct {
	nvar    int
	symbols map[string]*entry
	op      string
}

func NewBinder() *Binder {
	return &Binder{symbols: make(map[string]*entry)}
}

func (b *Binder) Define(name string, ref any) {
	b.symbols[name] = &entry{ref: ref}
}

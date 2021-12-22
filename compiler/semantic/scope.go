package semantic

import (
	"fmt"

	"github.com/brimdata/zed"
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

func (s *Scope) Enter() {
	s.stack = append(s.stack, NewBinder())
}

func (s *Scope) Exit() {
	s.stack = s.stack[:len(s.stack)-1]
}

func (s *Scope) DefineVar(name string) (*dag.Var, error) {
	b := s.tos()
	if _, ok := b.symbols[name]; ok {
		return nil, fmt.Errorf("symbol %q redefined", name)
	}
	ref := &dag.Var{
		Kind: "Var",
		Slot: s.nvars(),
	}
	b.Define(name, ref)
	b.nvar++
	return ref, nil
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
		return fmt.Errorf("cannot resolve const %q at compile time", name)
	}
	literal := &dag.Literal{
		Kind:  "Literal",
		Value: zson.MustFormatValue(*val),
	}
	b.Define(name, literal)
	return nil
}

func (s *Scope) Lookup(name string) dag.Expr {
	for k := len(s.stack) - 1; k >= 0; k-- {
		if e, ok := s.stack[k].symbols[name]; ok {
			e.refcnt++
			return e.ref
		}
	}
	return nil
}

func (s *Scope) nvars() int {
	var n int
	for k := len(s.stack) - 1; k >= 0; k-- {
		n += s.stack[k].nvar
	}
	return n
}

type entry struct {
	ref    dag.Expr
	refcnt int
}

type Binder struct {
	nvar    int
	symbols map[string]*entry
}

func NewBinder() *Binder {
	return &Binder{symbols: make(map[string]*entry)}
}

func (b *Binder) Define(name string, ref dag.Expr) {
	b.symbols[name] = &entry{ref: ref}
}

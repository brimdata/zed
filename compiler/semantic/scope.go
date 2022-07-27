package semantic

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
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

func (s *Scope) DefineConst(name string, def dag.Expr) error {
	b := s.tos()
	if _, ok := b.symbols[name]; ok {
		return fmt.Errorf("symbol %q redefined", name)
	}
	val, err := evalAtCompileTime(s.zctx, def) //XXX
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
	for _, scope := range s.stack {
		n += scope.nvar
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

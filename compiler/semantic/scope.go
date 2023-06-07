package semantic

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/kernel"
	"github.com/brimdata/zed/zson"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

type Scope struct {
	parent   *Scope
	children []*Scope
	nvar     int
	symbols  map[string]*entry
}

func NewScope(parent *Scope) *Scope {
	s := &Scope{parent: parent, symbols: make(map[string]*entry)}
	if parent != nil {
		parent.children = append(parent.children, s)
	}
	return s
}

type entry struct {
	ref    any
	refcnt int
	order  int
}

func (s *Scope) DefineVar(name string) error {
	ref := &dag.Var{
		Kind: "Var",
		Name: name,
		Slot: s.nvars(),
	}
	if err := s.DefineAs(name, ref); err != nil {
		return err
	}
	s.nvar++
	return nil
}

func (s *Scope) DefineAs(name string, e any) error {
	if _, ok := s.symbols[name]; ok {
		return fmt.Errorf("symbol %q redefined", name)
	}
	s.symbols[name] = &entry{ref: e, order: len(s.symbols)}
	return nil
}

func (s *Scope) DefineConst(zctx *zed.Context, name string, def dag.Expr) error {
	val, err := kernel.EvalAtCompileTime(zctx, def)
	if err != nil {
		return err
	}
	if val.IsError() {
		if val.IsMissing() {
			return fmt.Errorf("const %q: cannot have variable dependency", name)
		} else {
			return fmt.Errorf("const %q: %q", name, string(val.Bytes()))
		}
	}
	literal := &dag.Literal{
		Kind:  "Literal",
		Value: zson.MustFormatValue(val),
	}
	s.DefineAs(name, literal)
	return nil
}

func (s *Scope) LookupExpr(name string) (dag.Expr, error) {
	if entry := s.lookupEntry(name); entry != nil {
		e, ok := entry.ref.(dag.Expr)
		if !ok {
			return nil, fmt.Errorf("symbol %q is not bound to an expression", name)
		}
		entry.refcnt++
		return e, nil
	}
	return nil, nil
}

func (s *Scope) LookupOp(name string) (*dag.UserOp, error) {
	if entry := s.lookupEntry(name); entry != nil {
		d, ok := entry.ref.(*dag.UserOp)
		if !ok {
			return nil, fmt.Errorf("symbol %q is not bound to an operator", name)
		}
		entry.refcnt++
		return d, nil
	}
	return nil, nil
}

func (s *Scope) lookupEntry(name string) *entry {
	for scope := s; scope != nil; scope = scope.parent {
		if entry, ok := scope.symbols[name]; ok {
			return entry
		}
	}
	return nil
}

func (s *Scope) nvars() int {
	var n int
	for scope := s; scope != nil; scope = scope.parent {
		n += scope.nvar
	}
	return n
}

func (s *Scope) sortedEntries() []*entry {
	entries := maps.Values(s.symbols)
	slices.SortFunc(entries, func(i, j *entry) bool {
		return i.order < j.order
	})
	return entries
}

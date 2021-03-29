package semantic

import (
	"github.com/brimdata/zq/compiler/ast"
)

type Scope struct {
	stack []Binder
}

func NewScope() *Scope {
	return &Scope{}
}

func (s *Scope) tos() Binder {
	return s.stack[len(s.stack)-1]
}

func (s *Scope) Enter() {
	s.stack = append(s.stack, NewBinder())
}

func (s *Scope) Exit() {
	s.stack = s.stack[:len(s.stack)-1]
}

func (s *Scope) Bind(name string, ref ast.Proc) {
	s.tos().Define(name, ref)
}

func (s *Scope) Lookup(name string) ast.Proc {
	for k := len(s.stack) - 1; k >= 0; k-- {
		if e, ok := s.stack[k][name]; ok {
			e.refcnt++
			return e.proc
		}
	}
	return nil
}

type entry struct {
	proc   ast.Proc
	refcnt int
}

type Binder map[string]*entry

func NewBinder() Binder {
	return make(map[string]*entry)
}

func (b Binder) Define(name string, ref ast.Proc) {
	b[name] = &entry{proc: ref}
}

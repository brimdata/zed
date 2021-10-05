package kernel

import "github.com/brimdata/zed"

// A Scope is a stack of bindings that map identifiers to literals,
// generator variables, functions etc.  Currently, we only handle iterators
// but this will change soone as we add support for richer Zed script semantics.
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

func (s *Scope) Bind(name string, ref *zed.Value) {
	s.tos().Define(name, ref)
}

func (s *Scope) Lookup(name string) *zed.Value {
	for k := len(s.stack) - 1; k >= 0; k-- {
		if ref, ok := s.stack[k][name]; ok {
			return ref
		}
	}
	return nil
}

//XXX for now, Binder is just a map of identifiers to a specific zed.Value
// reference that the name refers to.  This will be generalized later to handle
// all possible types of identifier bindings.
type Binder map[string]*zed.Value

func NewBinder() Binder {
	return make(map[string]*zed.Value)
}

func (b Binder) Define(name string, ref *zed.Value) {
	b[name] = ref
}

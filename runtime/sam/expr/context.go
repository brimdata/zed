package expr

import (
	"github.com/brimdata/zed"
)

// Context is an interface to a scope and value allocator for expressions.
// This allows external packages to implement efficient temporary buffering
// of Zed values both for let-style temporary variables accessible via
// the scope and for allocating results.
type Context interface {
	Arena() *zed.Arena

	// Vars() accesses the variables reachable in the current scope.
	Vars() []zed.Value
}

type allocator struct {
	arena *zed.Arena
	vars  []zed.Value

	stackDepth int
}

var _ Context = (*allocator)(nil)

func NewContext(arena *zed.Arena) *allocator {
	return NewContextWithVars(arena, nil)
}

func NewContextWithVars(arena *zed.Arena, vars []zed.Value) *allocator {
	return &allocator{arena, vars, 0}
}

func (a *allocator) Arena() *zed.Arena {
	return a.arena
}

func (a *allocator) Vars() []zed.Value {
	return a.vars
}

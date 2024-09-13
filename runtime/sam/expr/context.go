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

type Ctx struct {
	arena *zed.Arena
	vars  []zed.Value
}

var _ Context = (*Ctx)(nil)

func NewContext(arena *zed.Arena) *Ctx {
	return NewContextWithVars(arena, nil)
}

func NewContextWithVars(arena *zed.Arena, vars []zed.Value) *Ctx {
	return &Ctx{arena, vars}
}

func (a *Ctx) Arena() *zed.Arena {
	return a.arena
}

func (a *Ctx) Vars() []zed.Value {
	return a.vars
}

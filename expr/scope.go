package expr

import (
	"github.com/brimdata/zed"
)

// Context is an interface to a scope and value allocator for expressions.
// This allows external packages to implement efficint temporary buffering
// of Zed values both for let-style temporary variables accessible via
// Context.Scope() and for allocating results.
type Context interface {
	Scope() []zed.Value
	//XXX there should be two NewValues: one when bytes is already inside
	// of the context... another when you need to copy those bytes into
	// this context.
	zed.Allocator
}

/*
type Scope []zed.Value

func (s Scope) Frame() []zed.Value {
	return s
}

func (s *Scope) Pop(n int) {
	*s = (*s)[:len(*s)-n]
}

func (s *Scope) Push(val zed.Value) {
	*s = append(*s, val)
}
*/

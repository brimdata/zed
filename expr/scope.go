package expr

import (
	"github.com/brimdata/zed"
)

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

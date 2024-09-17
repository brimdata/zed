package op

import "github.com/brimdata/zed/vector"

type Pass struct {
	parent vector.Puller
}

func NewPass(parent vector.Puller) *Pass {
	return &Pass{parent}
}

func (p *Pass) Pull(done bool) (vector.Any, error) {
	return p.parent.Pull(done)
}

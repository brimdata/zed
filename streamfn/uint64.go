package streamfn

import (
	"math"
)

type Uint64 struct {
	State  uint64
	Update func(uint64)
}

func NewUint64(op string) *Uint64 {
	p := &Uint64{}
	switch op {
	case "sum":
		p.Update = func(v uint64) {
			p.State += v
		}
	case "min":
		p.State = math.MaxUint64
		p.Update = func(v uint64) {
			if v < p.State {
				p.State = v
			}
		}
	case "max":
		p.Update = func(v uint64) {
			if v > p.State {
				p.State = v
			}
		}
	default:
		return nil
	}
	return p
}

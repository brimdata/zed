package streamfn

import (
	"math"
)

type Int64 struct {
	State  int64
	Update func(int64)
}

func NewInt64(op string) *Int64 {
	p := &Int64{}
	switch op {
	case "Sum":
		p.Update = func(v int64) {
			p.State += v
		}
	case "Min":
		p.State = math.MaxInt64
		p.Update = func(v int64) {
			if v < p.State {
				p.State = v
			}
		}
	case "Max":
		p.State = math.MinInt64
		p.Update = func(v int64) {
			if v > p.State {
				p.State = v
			}
		}
	default:
		return nil
	}
	return p
}

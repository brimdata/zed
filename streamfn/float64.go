package streamfn

import (
	"math"
)

type Float64 struct {
	State  float64
	Update func(float64)
}

func NewFloat64(op string) *Float64 {
	p := &Float64{}
	switch op {
	case "Sum":
		p.Update = func(v float64) {
			p.State += v
		}
	case "Min":
		p.State = math.MaxFloat64
		p.Update = func(v float64) {
			if v < p.State {
				p.State = v
			}
		}
	case "Max":
		p.State = -math.MaxFloat64
		p.Update = func(v float64) {
			if v > p.State {
				p.State = v
			}
		}
	default:
		return nil
	}
	return p
}

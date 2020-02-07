package streamfn

import (
	"math"

	"github.com/brimsec/zq/pkg/nano"
)

type Time struct {
	State  nano.Ts
	Update func(nano.Ts)
}

func NewTime(op string) *Time {
	p := &Time{}
	switch op {
	case "Sum":
		// XXX doesn't really make sense to sum absoute times
		p.Update = func(v nano.Ts) {
			p.State += v
		}
	case "Min":
		p.State = math.MaxInt64
		p.Update = func(v nano.Ts) {
			if v < p.State {
				p.State = v
			}
		}
	case "Max":
		p.State = math.MinInt64
		p.Update = func(v nano.Ts) {
			if v > p.State {
				p.State = v
			}
		}
	default:
		return nil
	}
	return p
}

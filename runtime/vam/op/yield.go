package op

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/vam/expr"
	"github.com/brimdata/zed/vector"
)

type Yield struct {
	zctx   *zed.Context
	parent vector.Puller
	exprs  []expr.Evaluator
}

var _ vector.Puller = (*Yield)(nil)

func NewYield(zctx *zed.Context, parent vector.Puller, exprs []expr.Evaluator) *Yield {
	return &Yield{
		zctx:   zctx,
		parent: parent,
		exprs:  exprs,
	}
}

func (y *Yield) Pull(done bool) (vector.Any, error) {
	for {
		val, err := y.parent.Pull(done)
		if val == nil {
			return nil, err
		}
		vals := make([]vector.Any, 0, len(y.exprs))
		for _, e := range y.exprs {
			v := filterQuiet(e.Eval(val))
			if v != nil {
				vals = append(vals, v)
			}
		}
		if len(vals) == 1 {
			return vals[0], nil
		} else if len(vals) != 0 {
			return interleave(vals), nil
		}
		// If no vals, continue the loop.
	}
}

func filterQuiet(val vector.Any) vector.Any {
	// XXX this can't happen until we have functions
	return val
}

// XXX should work for input variants
func interleave(vals []vector.Any) vector.Any {
	if len(vals) < 2 {
		panic("interleave requires two or more vals")
	}
	n := vals[0].Len()
	nvals := uint32(len(vals))
	tags := make([]uint32, n*nvals)
	for k := uint32(0); k < n*nvals; k++ {
		tags[k] = k % nvals

	}
	return vector.NewVariant(tags, vals)
}

package switcher

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zbuf"
)

type Selector struct {
	*op.Router
	cases []*switchCase
}

var _ op.Selector = (*Selector)(nil)

type switchCase struct {
	filter expr.Evaluator
	route  zbuf.Puller
	vals   []zed.Value
}

func New(pctx *op.Context, parent zbuf.Puller) *Selector {
	router := op.NewRouter(pctx, parent)
	s := &Selector{
		Router: router,
	}
	router.Link(s)
	return s
}

func (s *Selector) AddCase(f expr.Evaluator) zbuf.Puller {
	route := s.Router.AddRoute()
	s.cases = append(s.cases, &switchCase{filter: f, route: route})
	return route
}

func (s *Selector) Forward(router *op.Router, batch zbuf.Batch) bool {
	vals := batch.Values()
	for i := range vals {
		this := &vals[i]
		for _, c := range s.cases {
			val := c.filter.Eval(batch, this)
			if val.IsMissing() {
				continue
			}
			if val.IsError() {
				// XXX should use structured here to wrap
				// the input value with the error
				c.vals = append(c.vals, *val)
				continue
				//XXX don't break here?
				//break
			}
			if val.Type == zed.TypeBool && val.Bytes != nil && zed.IsTrue(val.Bytes) {
				c.vals = append(c.vals, *this)
				break
			}
		}
	}
	// Send each case that has vals from this batch.
	// We have vals that point into the current batch so we
	// ref the batch for each outgoing new batch.
	for _, c := range s.cases {
		if len(c.vals) > 0 {
			// XXX The new slice should come from the
			// outgoing batch so we don't send these slices
			// through GC.
			batch.Ref()
			out := zbuf.NewArray(c.vals)
			c.vals = nil
			if ok := router.Send(c.route, out, nil); !ok {
				return false
			}
		}
	}
	return true
}

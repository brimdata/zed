package switcher

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/runtime/sam/expr"
	"github.com/brimdata/zed/runtime/sam/op"
	"github.com/brimdata/zed/zbuf"
)

type Selector struct {
	*op.Router
	zctx  *zed.Context
	cases []*switchCase
}

var _ op.Selector = (*Selector)(nil)

type switchCase struct {
	filter expr.Evaluator
	route  zbuf.Puller
	vals   []zed.Value
}

func New(rctx *runtime.Context, parent zbuf.Puller) *Selector {
	router := op.NewRouter(rctx, parent)
	s := &Selector{
		Router: router,
		zctx:   rctx.Zctx,
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
	arena := zed.NewArena()
	defer arena.Unref()
	ectx := expr.NewContextWithVars(arena, batch.Vars())
	for _, this := range batch.Values() {
		for _, c := range s.cases {
			val := c.filter.Eval(ectx, this)
			if val.IsMissing() {
				continue
			}
			if val.IsError() {
				// XXX should use structured here to wrap
				// the input value with the error
				c.vals = append(c.vals, val)
				continue
				//XXX don't break here?
				//break
			}
			if val.Type() == zed.TypeBool && val.Bool() {
				c.vals = append(c.vals, this)
				break
			}
		}
	}
	// Send each case that has vals from this batch.
	// We have vals that point into the current batch so we
	// ref the batch for each outgoing new batch.
	for _, c := range s.cases {
		if len(c.vals) > 0 {
			out := zbuf.NewBatch(arena, c.vals, batch, batch.Vars())
			c.vals = nil
			if ok := router.Send(c.route, out, nil); !ok {
				return false
			}
		}
	}
	return true
}

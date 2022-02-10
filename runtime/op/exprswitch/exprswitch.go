package exprswitch

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zbuf"
)

type ExprSwitch struct {
	*op.Router
	expr        expr.Evaluator
	cases       map[string]*switchCase
	defaultCase *switchCase
}

var _ op.Selector = (*ExprSwitch)(nil)

type switchCase struct {
	route zbuf.Puller
	vals  []zed.Value
}

func New(pctx *op.Context, parent zbuf.Puller, e expr.Evaluator) *ExprSwitch {
	router := op.NewRouter(pctx, parent)
	s := &ExprSwitch{
		Router: router,
		expr:   e,
		cases:  make(map[string]*switchCase),
	}
	router.Link(s)
	return s
}

func (s *ExprSwitch) AddCase(val *zed.Value) zbuf.Puller {
	route := s.Router.AddRoute()
	if val == nil {
		s.defaultCase = &switchCase{route: route}
	} else {
		s.cases[string(val.Bytes)] = &switchCase{route: route}
	}
	return route
}

func (s *ExprSwitch) Forward(router *op.Router, batch zbuf.Batch) bool {
	vals := batch.Values()
	for i := range vals {
		val := s.expr.Eval(batch, &vals[i])
		if val.IsMissing() {
			continue
		}
		which, ok := s.cases[string(val.Bytes)]
		if !ok {
			which = s.defaultCase
		}
		if which == nil {
			continue
		}
		which.vals = append(which.vals, vals[i])
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
	if c := s.defaultCase; c != nil && len(c.vals) > 0 {
		batch.Ref()
		out := zbuf.NewArray(c.vals)
		c.vals = nil
		if ok := router.Send(c.route, out, nil); !ok {
			return false
		}
	}
	return true
}

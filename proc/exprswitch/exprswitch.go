package exprswitch

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
)

type ExprSwitch struct {
	*proc.Router
	expr        expr.Evaluator
	cases       map[string]*switchCase
	defaultCase *switchCase
}

var _ proc.Selector = (*ExprSwitch)(nil)

type switchCase struct {
	route proc.Interface
	vals  []zed.Value
}

func New(pctx *proc.Context, parent proc.Interface, e expr.Evaluator) *ExprSwitch {
	router := proc.NewRouter(pctx, parent)
	s := &ExprSwitch{
		Router: router,
		expr:   e,
		cases:  make(map[string]*switchCase),
	}
	router.Link(s)
	return s
}

func (s *ExprSwitch) AddCase(val *zed.Value) proc.Interface {
	route := s.Router.AddRoute()
	if val == nil {
		s.defaultCase = &switchCase{route: route}
	} else {
		s.cases[string(val.Bytes)] = &switchCase{route: route}
	}
	return route
}

func (s *ExprSwitch) Forward(router *proc.Router, batch zbuf.Batch) error {
	ectx := batch.Context()
	vals := batch.Values()
	for i := range vals {
		val := s.expr.Eval(ectx, &vals[i])
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
	// ref the batch for each outgoing new batch, then unref
	// the batch once after the loop.
	for _, c := range s.cases {
		if len(c.vals) > 0 {
			// XXX The new slice should come from the
			// outgoing batch so we don't send these slices
			// through GC.
			batch.Ref()
			out := zbuf.NewArray(c.vals)
			c.vals = nil
			router.Send(c.route, out, nil)
		}
	}
	if c := s.defaultCase; c != nil && len(c.vals) > 0 {
		batch.Ref()
		out := zbuf.NewArray(c.vals)
		c.vals = nil
		router.Send(c.route, out, nil)
	}
	return nil
}

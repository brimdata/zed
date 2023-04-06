package fork

import (
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zbuf"
)

type Op struct {
	router *op.Router
	exits  []zbuf.Puller
}

func New(octx *op.Context, parent zbuf.Puller) *Op {
	return &Op{router: op.NewRouter(octx, parent)}
}

func (o *Op) AddExit() zbuf.Puller {
	exit := o.router.AddRoute()
	o.exits = append(o.exits, exit)
	// Calling Link repeatedly is safe.
	o.router.Link(splitter(o.exits))
	return exit
}

// A splitter splits its input into multiple output operators by implementing
// op.Selector and selecting all downstream legs of the flowgraph.
type splitter []zbuf.Puller

var _ op.Selector = (*splitter)(nil)

// Forward copies every batch to every output thus implementing fork.
func (s splitter) Forward(r *op.Router, b zbuf.Batch) bool {
	for _, exit := range s {
		b.Ref()
		if ok := r.Send(exit, b, nil); !ok {
			return false
		}
	}
	return true
}

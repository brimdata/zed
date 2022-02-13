package fork

import (
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zbuf"
)

// A splitter splits its input into multiple proc outputs by implementing
// op.Selector and selecting all downstream legs of the flowgraph.
type splitter []zbuf.Puller

var _ op.Selector = (*splitter)(nil)

func New(pctx *op.Context, parent zbuf.Puller, n int) []zbuf.Puller {
	router := op.NewRouter(pctx, parent)
	exits := make([]zbuf.Puller, 0, n)
	for k := 0; k < n; k++ {
		exits = append(exits, router.AddRoute())
	}
	router.Link(splitter(exits))
	return exits
}

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

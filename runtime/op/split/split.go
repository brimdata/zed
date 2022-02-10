package split

import (
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zbuf"
)

// Splitter splits its input into multiple proc outputs.  Since procs run from the
// receiver backward via Pull(), SplitProc pulls data from upstream when all the
// outputs are ready, then sends the data downstream.
//
// This scheme implements flow control since the SplitProc prevents any of
// the downstream from running ahead, esentially running the parallel paths
// at the rate of the slowest consumer.
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

// Forward copies every batch to every output thus implementing split.
func (s splitter) Forward(r *op.Router, b zbuf.Batch) bool {
	for _, exit := range s {
		b.Ref()
		if ok := r.Send(exit, b, nil); !ok {
			return false
		}
	}
	return true
}

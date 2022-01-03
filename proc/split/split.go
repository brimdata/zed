package split

import (
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
)

// Splitter splits its input into multiple proc outputs.  Since procs run from the
// receiver backward via Pull(), SplitProc pulls data from upstream when all the
// outputs are ready, then sends the data downstream.
//
// This scheme implements flow control since the SplitProc prevents any of
// the downstream from running ahead, esentially running the parallel paths
// at the rate of the slowest consumer.
type splitter []proc.Interface

var _ proc.Selector = (*splitter)(nil)

func New(pctx *proc.Context, parent proc.Interface, n int) []proc.Interface {
	router := proc.NewRouter(pctx, parent)
	exits := make([]proc.Interface, 0, n)
	for k := 0; k < n; k++ {
		exits = append(exits, router.AddRoute())
	}
	router.Link(splitter(exits))
	return exits
}

// Forward copies every batch to every output thus implementing split.
func (s splitter) Forward(r *proc.Router, b zbuf.Batch) error {
	for _, exit := range s {
		b.Ref()
		r.Send(exit, b, nil)
	}
	return nil
}

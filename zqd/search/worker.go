package search

import (
	"context"
	"fmt"
	"time"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/driver"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/storage"
	"github.com/brimsec/zq/zqd/storage/archivestore"
	"github.com/brimsec/zq/zqe"
)

type WorkerOp struct {
	ss   archive.SpanInfoSource
	proc ast.Proc
	dir  int
}

func NewWorkerOp(req api.WorkerRequest) (*WorkerOp, error) {
	// XXX zqd only supports backwards searches, remove once this has been
	// fixed.
	if req.Dir == 1 {
		return nil, zqe.E(zqe.Invalid, "forward searches not yet supported")
	}
	if req.Dir != -1 {
		return nil, zqe.E(zqe.Invalid, "time direction must be 1 or -1")
	}

	proc, err := ast.UnpackJSON(nil, req.Proc)
	if err != nil {
		return nil, err
	}

	ss := archive.SpanInfoSource{
		Span:       req.Span,
		ChunkPaths: req.ChunkPaths,
	}
	return &WorkerOp{ss: ss, proc: proc, dir: req.Dir}, nil
}

func (w *WorkerOp) Run(ctx context.Context, store storage.Storage, output Output) (err error) {
	d := &searchdriver{
		output:    output,
		startTime: nano.Now(),
	}
	d.start(0)
	defer func() {
		if err != nil {
			d.abort(0, err)
			return
		}
		d.end(0)
	}()

	statsTicker := time.NewTicker(StatsInterval)
	defer statsTicker.Stop()
	zctx := resolver.NewContext()

	switch st := store.(type) {
	case *archivestore.Storage:
		return driver.MultiRun(ctx, d, w.proc, zctx, st.StaticSource(w.ss), driver.MultiConfig{
			StatsTick: statsTicker.C,
		})
	default:
		return fmt.Errorf("unknown storage type %T", st)
	}
}

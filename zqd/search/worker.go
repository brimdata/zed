package search

import (
	"context"
	"time"

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
	proc  ast.Proc
	span  nano.Span
	src   driver.Source
	store *archivestore.Storage
}

func NewWorkerOp(ctx context.Context, req api.WorkerRequest, st storage.Storage) (*WorkerOp, error) {
	// XXX zqd only supports backwards searches, remove once this has been
	// fixed.
	if req.Dir == 1 {
		return nil, zqe.E(zqe.Invalid, "worker: forward searches not yet supported")
	}
	if req.Dir != -1 {
		return nil, zqe.E(zqe.Invalid, "worker: time direction must be 1 or -1")
	}
	store, ok := st.(*archivestore.Storage)
	if !ok {
		return nil, zqe.ErrInvalid("unhandled storage type for WorkerOp: %T", store)
	}
	src, err := store.MultiSource().SourceFromRequest(ctx, &req)
	if err != nil {
		return nil, zqe.ErrInvalid("invalid worker op request: %w", err)
	}
	proc, err := ast.UnpackJSON(nil, req.Proc)
	if err != nil {
		return nil, err
	}
	return &WorkerOp{proc: proc, span: req.Span, src: src, store: store}, nil
}

func (w *WorkerOp) Run(ctx context.Context, dir int, output Output) (err error) {
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

	return driver.MultiRun(ctx, d, w.proc, zctx, w.store.StaticSource(w.src), driver.MultiConfig{
		Span:        w.span,
		StatsTick:   statsTicker.C,
		Dir:         dir,
		Parallelism: 1,
	})
}

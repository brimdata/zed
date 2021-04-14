package search

import (
	"context"
	"time"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/ppl/zqd/storage"
	"github.com/brimdata/zed/ppl/zqd/storage/archivestore"
	"github.com/brimdata/zed/zqe"
	"github.com/brimdata/zed/zson"
	"go.uber.org/zap"
)

type WorkerOp struct {
	logger *zap.Logger
	proc   ast.Proc
	span   nano.Span
	src    driver.Source
	store  *archivestore.Storage
}

func NewWorkerOp(ctx context.Context, req api.WorkerChunkRequest, st storage.Storage, logger *zap.Logger) (*WorkerOp, error) {
	// XXX zqd only supports backwards searches, remove once this has been
	// fixed.
	if req.Dir == 1 {
		return nil, zqe.E(zqe.Invalid, "forward searches not yet supported")
	}
	if req.Dir != -1 {
		return nil, zqe.E(zqe.Invalid, "time direction must be 1 or -1")
	}
	store, ok := st.(*archivestore.Storage)
	if !ok {
		return nil, zqe.ErrInvalid("unhandled storage type for WorkerOp: %T", store)
	}
	src, err := store.MultiSource().SourceFromRequest(ctx, &req)
	if err != nil {
		return nil, zqe.ErrInvalid("invalid worker op request: %w", err)
	}
	proc, err := ast.UnpackJSONAsProc(req.Proc)
	if err != nil {
		return nil, err
	}
	return &WorkerOp{
		logger: logger,
		proc:   proc,
		span:   req.Span,
		src:    src,
		store:  store,
	}, nil
}

func (w *WorkerOp) Run(ctx context.Context, output Output) (err error) {
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

	return driver.MultiRun(ctx, d, w.proc, zson.NewContext(), w.store.StaticSource(w.src), driver.MultiConfig{
		Logger:    w.logger,
		Span:      w.span,
		StatsTick: statsTicker.C,
	})
}

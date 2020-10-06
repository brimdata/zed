// Package search provides an implementation for launching zq searches and performing
// analytics on zng files stored in the server's root directory.
package search

import (
	"context"
	"fmt"
	"time"

	"github.com/brimsec/zq/archive"
	"github.com/segmentio/ksuid"

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
	si   archive.SpanInfo
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
	//query, err := UnpackQuery(req)

	id, err := ksuid.Parse(req.ChunkId)
	if err != nil {
		return nil, zqe.E(zqe.Invalid, "unparsable ksuid")
	}

	chunk := archive.Chunk{
		Id:           id,
		First:        req.ChunkFirst,
		Last:         req.ChunkLast,
		DataFileKind: archive.FileKind(req.ChunkFileKind),
		RecordCount:  req.ChunkRecordCount,
	}
	si := archive.SpanInfo{
		Span:   chunk.Span(), // make a Span from the chunk
		Chunks: []archive.Chunk{chunk},
	}

	proc, err := ast.UnpackJSON(nil, req.Proc)
	if err != nil {
		return nil, err
	}

	return &WorkerOp{si: si, proc: proc, dir: req.Dir}, nil
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
		return driver.MultiRun(ctx, d, w.proc, zctx, st.StaticSource(w.si), driver.MultiConfig{
			Span:      w.si.Span,
			StatsTick: statsTicker.C,
		})
	default:
		return fmt.Errorf("unknown storage type %T", st)
	}
}

package zqd

import (
	"net/http"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/ppl/archive"
	"github.com/brimsec/zq/ppl/zqd/search"
	"github.com/brimsec/zq/ppl/zqd/storage/archivestore"
	"github.com/brimsec/zq/zqe"
	"go.uber.org/zap"
)

func handleWorkerRootSearch(c *Core, w http.ResponseWriter, r *http.Request) {
	var req api.WorkerRootRequest
	if !request(c, w, r, &req) {
		return
	}
	if req.MaxWorkers < 1 || req.MaxWorkers > 100 {
		// Limit is hard coded for now to 100 for initial testing and research.
		err := zqe.ErrInvalid("number of workers requested must be between 1 and 100")
		respondError(c, w, r, err)
		return
	}
	srch, err := search.NewSearchOp(req.SearchRequest)
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	out, err := getSearchOutput(w, r)
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	store, err := c.mgr.GetStorage(r.Context(), req.SearchRequest.Space)
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	w.Header().Set("Content-Type", out.ContentType())
	if err := srch.RunDistributed(r.Context(), store, out, req.MaxWorkers, c.worker, c.logger); err != nil {
		c.requestLogger(r).Warn("Error writing response", zap.Error(err))
	}
}

func handleWorkerChunkSearch(c *Core, w http.ResponseWriter, httpReq *http.Request) {
	defer c.workerReg.SendRelease()
	var req api.WorkerChunkRequest
	if !request(c, w, httpReq, &req) {
		return
	}
	ctx := httpReq.Context()
	ark, err := archive.OpenArchiveWithContext(ctx, req.DataPath, &archive.OpenOptions{})
	if err != nil {
		respondError(c, w, httpReq, err)
		return
	}
	work, err := search.NewWorkerOp(ctx, req, archivestore.NewStorage(ark))
	if err != nil {
		respondError(c, w, httpReq, err)
		return
	}
	out, err := getSearchOutput(w, httpReq)
	if err != nil {
		respondError(c, w, httpReq, err)
		return
	}
	w.Header().Set("Content-Type", out.ContentType())
	if err := work.Run(ctx, out); err != nil {
		c.requestLogger(httpReq).Warn("Error writing response", zap.Error(err))
	}
}

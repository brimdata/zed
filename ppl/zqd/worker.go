package zqd

import (
	"net/http"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/ppl/zqd/search"
	"github.com/brimdata/zed/ppl/zqd/storage/archivestore"
	"github.com/brimdata/zed/zqe"
	"github.com/brimdata/zed/zson"
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
	srch, err := search.NewSearchOp(req.SearchRequest, c.requestLogger(r))
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	zctx := zson.NewContext()
	out, err := getSearchOutput(w, r, zctx)
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
	if err := srch.Run(r.Context(), store, out, req.MaxWorkers, c.conf.Worker, zctx); err != nil {
		c.requestLogger(r).Warn("Error writing response", zap.Error(err))
	}
}

func handleWorkerChunkSearch(c *Core, w http.ResponseWriter, httpReq *http.Request) {
	c.workerReg.SendBusy()
	defer c.workerReg.SendIdle()
	var req api.WorkerChunkRequest
	if !request(c, w, httpReq, &req) {
		return
	}
	ctx := httpReq.Context()
	lk, err := lake.OpenLakeWithContext(ctx, req.DataPath, &lake.OpenOptions{})
	if err != nil {
		respondError(c, w, httpReq, err)
		return
	}
	work, err := search.NewWorkerOp(ctx, req, archivestore.NewStorage(lk), c.requestLogger(httpReq))
	if err != nil {
		respondError(c, w, httpReq, err)
		return
	}
	zctx := zson.NewContext()
	out, err := getSearchOutput(w, httpReq, zctx)
	if err != nil {
		respondError(c, w, httpReq, err)
		return
	}
	w.Header().Set("Content-Type", out.ContentType())
	if err := work.Run(ctx, out, zctx); err != nil {
		c.requestLogger(httpReq).Warn("Error writing response", zap.Error(err))
	}
}

func handleWorkerRelease(c *Core, w http.ResponseWriter, httpReq *http.Request) {
	c.workerReg.SendRelease()
	w.WriteHeader(http.StatusNoContent)
}

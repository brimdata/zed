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
	println("ROOT SEARCH")
	var req api.WorkerRootRequest
	if !request(c, w, r, &req) {
		return
	}

	if req.MaxWorkers < 1 || req.MaxWorkers > 100 {
		// Note: the upper limit of distributed workers for any given query
		// will be determined based on future testing.
		// Hard coded for now to 100 for intial testing and research.
		// It will be based the size of the worker cluster, and will be
		// included in the environment.
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
	if err := srch.RunDistributed(r.Context(), store, out, req.MaxWorkers, c.worker.Recruiter, c.worker.BoundWorkers); err != nil {
		c.requestLogger(r).Warn("Error writing response", zap.Error(err))
	}
}

// handleWorkerChunkSearch implements the API used for
// distributed queries from a root worker process.
// Distributed query worker processes, after being
// recruited by a root worker, expect to recieve
// a series of chunk search requests, followed by a
// worker release request. If this series is interupted,
// the worker could become a "zombie" process that will not be
// recruited and is not recieving requests.
// The "zombie timer" system provides a way for workers
// in this state to exit.
func handleWorkerChunkSearch(c *Core, w http.ResponseWriter, httpReq *http.Request) {
	println("CHUNK SEARCH")
	// Insure workers perform one search at a time
	c.workerReg.SearchLock.Lock()
	defer c.workerReg.SearchLock.Unlock()
	println("CHUNK SEARCH inside lock")
	// Workers performing a search cannot be zombies.
	c.workerReg.StopZombieTimer()
	defer c.workerReg.StartZombieTimer()
	println("CHUNK SEARCH inside timers")

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

	c.workerReg.StartZombieTimer()
}

// handleWorkerRelease handles the /worker/release request that the
// zqd root process calls when it has completed a search query and
// intends to release the workers for use by another root process.
func handleWorkerRelease(c *Core, w http.ResponseWriter, httpReq *http.Request) {
	c.workerReg.StopZombieTimer()
	w.WriteHeader(http.StatusNoContent)
	go c.workerReg.RegisterWithRecruiter()
}

package zqd

import (
	"encoding/json"
	"net/http"

	"github.com/brimdata/zq/api"
	"github.com/brimdata/zq/ppl/zqd/recruiter"
	"github.com/brimdata/zq/zqe"
	"go.uber.org/zap"
)

// handleRecruit and handleRegister interact with each other:
// completing a request to handleRecruit will unblock multiple
// open requests (long polls) to handleRegister.
func handleRecruit(c *Core, w http.ResponseWriter, r *http.Request) {
	var req api.RecruitRequest
	if !request(c, w, r, &req) {
		return
	}
	workers, err := recruiter.RecruitWithEndWait(c.workerPool, req.NumberRequested, req.Label, c.logger)
	if err != nil {
		respondError(c, w, r, zqe.ErrInvalid(err))
		return
	}
	respond(c, w, r, http.StatusOK, api.RecruitResponse{
		Workers: workers,
	})
}

func handleRegister(c *Core, w http.ResponseWriter, r *http.Request) {
	var req api.RegisterRequest
	if !request(c, w, r, &req) {
		return
	}
	if req.Timeout <= 0 {
		respondError(c, w, r, zqe.E(zqe.Invalid, "required parameter timeout"))
		return
	}
	directive, cancelled, err := recruiter.WaitForRecruitment(r.Context(), c.workerPool,
		req.Addr, req.NodeName, req.Timeout, c.logger)
	if err != nil {
		respondError(c, w, r, zqe.ErrInvalid(err))
		return
	}
	if !cancelled {
		respond(c, w, r, http.StatusOK, api.RegisterResponse{Directive: directive})
	}
}

func handleRecruiterStats(c *Core, w http.ResponseWriter, r *http.Request) {
	respond(c, w, r, http.StatusOK, api.RecruiterStatsResponse{
		LenFreePool: c.workerPool.LenFreePool(),
		LenNodePool: c.workerPool.LenNodePool(),
	})
}

// handleListFree pretty prints the output because it is for manual trouble-shooting.
func handleListFree(c *Core, w http.ResponseWriter, r *http.Request) {
	ws := c.workerPool.ListFreePool()
	workers := make([]api.Worker, len(ws))
	for i, e := range ws {
		workers[i] = api.Worker{Addr: e.Addr, NodeName: e.NodeName}
	}
	body := api.RecruitResponse{
		Workers: workers,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(body); err != nil {
		c.requestLogger(r).Warn("Error writing response", zap.Error(err))
	}
}

package zqd

// system test with: make TEST=TestZq/ztests/suite/zqd/rec-curl

import (
	"encoding/json"
	"net/http"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/zqe"
	"go.uber.org/zap"
)

func handleDeregister(c *Core, w http.ResponseWriter, r *http.Request) {
	var req api.DeregisterRequest
	if !request(c, w, r, &req) {
		return
	}
	c.workerPool.Deregister(req.Addr)
	respond(c, w, r, http.StatusOK, api.RegisterResponse{
		Registered: false,
	})
}

func handleRecruit(c *Core, w http.ResponseWriter, r *http.Request) {
	var req api.RecruitRequest
	if !request(c, w, r, &req) {
		return
	}
	ws, err := c.workerPool.Recruit(req.NumberRequested)
	if err != nil {
		respondError(c, w, r, zqe.ErrInvalid(err))
		return
	}
	workers := make([]api.Worker, len(ws))
	for i, e := range ws {
		workers[i] = api.Worker{WorkerAddr: api.WorkerAddr{Addr: e.Addr}, NodeName: e.NodeName}
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
	registered, err := c.workerPool.Register(req.Addr, req.NodeName)
	if err != nil {
		respondError(c, w, r, zqe.ErrInvalid(err))
		return
	}
	respond(c, w, r, http.StatusOK, api.RegisterResponse{
		Registered: registered,
	})
}

func handleUnreserve(c *Core, w http.ResponseWriter, r *http.Request) {
	var req api.UnreserveRequest
	if !request(c, w, r, &req) {
		return
	}
	c.workerPool.Unreserve(req.Addrs)
	respond(c, w, r, http.StatusOK, api.UnreserveResponse{
		Reserved: false,
	})
}

func handleWorkersStats(c *Core, w http.ResponseWriter, r *http.Request) {
	respond(c, w, r, http.StatusOK, api.WorkersStatsResponse{
		LenFreePool:     c.workerPool.LenFreePool(),
		LenReservedPool: c.workerPool.LenReservedPool(),
		LenNodePool:     c.workerPool.LenNodePool(),
	})
}

// handleListFree pretty prints the output because it is for manual trouble-shooting
func handleListFree(c *Core, w http.ResponseWriter, r *http.Request) {
	ws := c.workerPool.ListFreePool()
	workers := make([]api.Worker, len(ws))
	for i, e := range ws {
		workers[i] = api.Worker{WorkerAddr: api.WorkerAddr{Addr: e.Addr}, NodeName: e.NodeName}
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

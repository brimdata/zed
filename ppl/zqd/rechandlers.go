package zqd

import (
	"net/http"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/zqe"
)

func handleRegister(c *Core, w http.ResponseWriter, r *http.Request) {
	var req api.RegisterRequest
	if !request(c, w, r, &req) {
		return
	}
	err := c.workerPool.Register(req.Addr, req.NodeName)
	if err != nil {
		respondError(c, w, r, zqe.ErrInvalid(err))
		return
	}
	respond(c, w, r, http.StatusOK, api.StatusResponse{
		Status: "ok",
	})
}

func handleDeregister(c *Core, w http.ResponseWriter, r *http.Request) {
	var req api.DeregisterRequest
	if !request(c, w, r, &req) {
		return
	}
	c.workerPool.Deregister(req.Addr)
	respond(c, w, r, http.StatusOK, api.StatusResponse{
		Status: "ok",
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
		workers[i] = api.Worker{Addr: e.Addr, NodeName: e.NodeName}
	}
	respond(c, w, r, http.StatusOK, api.RecruitResponse{
		Workers: workers,
	})
}

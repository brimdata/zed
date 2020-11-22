package zqd

// Note that the handlers in this file write errors to their ResponseWriter,
// so there are no errors returned from handle functions.

// Useful CLI tests for recruiter API:
// zqd listen -l=localhost:8020 -portfile=portfile -personality=recruiter
// curl --header "Content-Type: application/json" -request POST --data '{"N":2}' http://localhost:8020/recruit
// curl --header "Content-Type: application/json" -request POST --data '{"addr":"a.b.c:5000","node":"a.b"}' http://localhost:8020/register
// curl --header "Content-Type: application/json" -request POST --data '{"addr":"a.b.c:5000"}' http://localhost:8020/unreserve
// curl --header "Content-Type: application/json" -request POST --data '{"addr":"a.b.c:5000"}' http://localhost:8020/deregister
// Or run system test with: make TEST=TestZq/ztests/suite/zqd/rec-curl

import (
	"net/http"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/zqe"
)

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
	err := c.workerPool.Register(req.Addr, req.NodeName)
	if err != nil {
		respondError(c, w, r, zqe.ErrInvalid(err))
		return
	}
	respond(c, w, r, http.StatusOK, api.StatusResponse{
		Status: "ok",
	})
}

func handleUnreserve(c *Core, w http.ResponseWriter, r *http.Request) {
	var req api.UnreserveRequest
	if !request(c, w, r, &req) {
		return
	}
	c.workerPool.Unreserve(req.Addr)
	respond(c, w, r, http.StatusOK, api.StatusResponse{
		Status: "ok",
	})
}

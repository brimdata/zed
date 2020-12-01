package zqd

// Note that the handlers in this file write errors to their ResponseWriter,
// so there are no errors returned from handle functions.

// Useful CLI tests for recruiter API:
// zqd listen -l=localhost:8020 -personality=recruiter
// curl --header "Content-Type: application/json" -request POST --data '{"N":2}' http://localhost:8020/recruit
// curl --header "Content-Type: application/json" -request POST --data '{"addr":"a.b.c:5000","node":"a.b"}' http://localhost:8020/register
// curl --header "Content-Type: application/json" -request POST --data '{"addr":"a.b.c:5000"}' http://localhost:8020/unreserve
// curl --header "Content-Type: application/json" -request POST --data '{"addr":"a.b.c:5000"}' http://localhost:8020/deregister
// Or run system test with: make TEST=TestZq/ztests/suite/zqd/rec-curl
// To test the API using the connection from a worker zqd to the recruiter, start a worker with:
// ZQD_NODE_NAME=mytest zqd listen -l=localhost:8030 -recruiter=localhost:8020

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
	//println(time.Now().Unix()%10000, "/workers/recruit", req.NumberRequested)
	workers := make([]api.Worker, len(ws))
	for i, e := range ws {
		workers[i] = api.Worker{WorkerAddr: api.WorkerAddr{Addr: e.Addr}, NodeName: e.NodeName}
		//println(e.Addr, e.NodeName)
	}
	//println("done")
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
	//println(time.Now().Unix()%10000, "/workers/register", req.Addr, req.NodeName)
	respond(c, w, r, http.StatusOK, api.RegisterResponse{
		Registered: registered,
	})
}

func handleUnreserve(c *Core, w http.ResponseWriter, r *http.Request) {
	var req api.UnreserveRequest
	if !request(c, w, r, &req) {
		return
	}
	c.workerPool.Unreserve(req.Addr)
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

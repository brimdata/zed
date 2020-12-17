package zqd

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/ppl/zqd/recruiter"
	"github.com/brimsec/zq/zqe"
	"go.uber.org/zap"
)

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
	var workers []api.Worker
	for _, e := range ws {
		if e.Callback(recruiter.RecruitmentDetail{Label: req.Label, NumberRequested: req.NumberRequested}) {
			workers = append(workers, api.Worker{Addr: e.Addr, NodeName: e.NodeName})
		}
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
	timer := time.NewTimer(time.Duration(req.Timeout) * time.Millisecond)
	recruited := make(chan recruiter.RecruitmentDetail)
	defer timer.Stop()
	cb := func(rd recruiter.RecruitmentDetail) bool {
		// Stopping the timer narrows the window for a timeout before
		// writing RecruitmentDetail to the channel that is read in the
		// select below.
		timer.Stop()
		// Use a non-blocking write because this will be called
		// while workerPool.Recruit is holding the workerPool lock.
		select {
		case recruited <- rd:
		default:
			c.logger.Warn("receiver not ready for recruited", zap.String("label", rd.Label))
			return false
			// Note that this warning could be logged if the recruiter timer fires
			// very close to the same time as a /recruiter/recruit request is processed.
			// Returning false insures that the worker address is not be returned
			// to a root process that recruited.
		}
		return true
	}
	if err := c.workerPool.Register(req.Addr, req.NodeName, cb); err != nil {
		respondError(c, w, r, zqe.ErrInvalid(err))
		return
	}
	// directive is one of:
	//  "reserved"   indicates to the worker that is has been reserved by a root process.
	//  "reregister" indicates the request timed out without the worker being reserved,
	//               and the worker should send another register request.
	var directive string
	var isCanceled bool
	ctx := r.Context()
	select {
	case rd := <-recruited:
		c.requestLogger(r).Info("Worker recruited",
			zap.String("addr", req.Addr),
			zap.String("label", rd.Label),
			zap.Int("count", rd.NumberRequested))
		directive = "reserved"
	case <-timer.C:
		c.requestLogger(r).Info("Worker should reregister", zap.String("addr", req.Addr))
		directive = "reregister"
	case <-ctx.Done():
		c.requestLogger(r).Info("handleRegister context cancel")
		isCanceled = true
	}
	// Deregister in any event
	c.workerPool.Deregister(req.Addr)
	if !isCanceled {
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

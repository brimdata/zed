package zqd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/brimsec/zq/api"
	"github.com/brimsec/zq/pcap"
	"github.com/brimsec/zq/pkg/ctxio"
	"github.com/brimsec/zq/ppl/archive"
	"github.com/brimsec/zq/ppl/zqd/ingest"
	"github.com/brimsec/zq/ppl/zqd/jsonpipe"
	"github.com/brimsec/zq/ppl/zqd/search"
	"github.com/brimsec/zq/ppl/zqd/space"
	"github.com/brimsec/zq/ppl/zqd/storage/archivestore"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zqe"
	"github.com/brimsec/zq/zql"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func errorResponse(e error) (status int, ae *api.Error) {
	status = http.StatusInternalServerError
	ae = &api.Error{Type: "Error"}

	var ze *zqe.Error
	if !errors.As(e, &ze) {
		ae.Message = e.Error()
		return
	}

	switch ze.Kind {
	case zqe.Invalid:
		status = http.StatusBadRequest
	case zqe.NotFound:
		status = http.StatusNotFound
	case zqe.Exists:
		status = http.StatusBadRequest
	case zqe.Conflict:
		status = http.StatusConflict
	}

	ae.Kind = ze.Kind.String()
	ae.Message = ze.Message()
	return
}

func respond(c *Core, w http.ResponseWriter, r *http.Request, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		c.requestLogger(r).Warn("Error writing response", zap.Error(err))
	}
}

func respondError(c *Core, w http.ResponseWriter, r *http.Request, err error) {
	status, ae := errorResponse(err)
	if status >= 500 {
		c.requestLogger(r).Warn("error", zap.Int("status", status), zap.Error(err))
	}
	respond(c, w, r, status, ae)
}

func request(c *Core, w http.ResponseWriter, r *http.Request, apiobj interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(apiobj); err != nil {
		respondError(c, w, r, zqe.E(zqe.Invalid, err))
		return false
	}
	return true
}

func handleASTPost(c *Core, w http.ResponseWriter, r *http.Request) {
	var req api.ASTRequest
	if !request(c, w, r, &req) {
		return
	}
	proc, err := zql.ParseProc(req.ZQL)
	if err != nil {
		respondError(c, w, r, zqe.ErrInvalid(err))
		return
	}
	respond(c, w, r, http.StatusOK, proc)
}

func handleSearch(c *Core, w http.ResponseWriter, r *http.Request) {
	var req api.SearchRequest
	if !request(c, w, r, &req) {
		return
	}

	s, err := c.spaces.Get(req.Space)
	if err != nil {
		respondError(c, w, r, err)
		return
	}

	ctx, cancel, err := s.StartOp(r.Context())
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	defer cancel()

	srch, err := search.NewSearchOp(req)
	if err != nil {
		// XXX This always returns bad request but should return status codes
		// that reflect the nature of the returned error.
		respondError(c, w, r, err)
		return
	}

	out, err := getSearchOutput(w, r)
	if err != nil {
		respondError(c, w, r, err)
		return
	}

	var order zbuf.Order
	if req.Dir == -1 {
		order = zbuf.OrderDesc
	}

	w.Header().Set("Content-Type", out.ContentType())
	if err := srch.Run(ctx, order, s, out); err != nil {
		c.requestLogger(r).Warn("Error writing response", zap.Error(err))
	}
}

// func handleWorker(c *Core, w http.ResponseWriter, httpReq *http.Request) {
// 	if os.Getenv("ZQD_WORKER_ONCE") == "true" {
// 		// With this env flag, a worker process will only handle
// 		// one /worker request, after which it exits.
// 		// If it gets a second request it is an error.
// 		c.workerOnce.Do(func() {
// 			handleWorkerSearch(c, w, httpReq)
// 			println(time.Now().Unix()%10000, "handleWorkerSearch complete")
// 			//c.logger.Info("Scheduled exit after single /worker request")
// 			//os.Exit(0)
// 		})
// 		respondError(c, w, httpReq, fmt.Errorf("zqd process already busy with /worker request"))
// 	} else {
// 		handleWorkerSearch(c, w, httpReq)
// 	}
// }

func handleWorker(c *Core, w http.ResponseWriter, httpReq *http.Request) {
	var req api.WorkerRequest
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

func getSearchOutput(w http.ResponseWriter, r *http.Request) (search.Output, error) {
	ctrl := true
	if r.URL.Query().Get("noctrl") != "" {
		ctrl = false
	}
	format := r.URL.Query().Get("format")
	switch format {
	case "csv":
		return search.NewCSVOutput(w, ctrl), nil
	case "json":
		return search.NewJSONOutput(w, search.DefaultMTU, ctrl), nil
	case "ndjson":
		return search.NewNDJSONOutput(w), nil
	case "zjson":
		return search.NewZJSONOutput(w, search.DefaultMTU, ctrl), nil
	case "zng":
		return search.NewZngOutput(w, ctrl), nil
	default:
		return nil, zqe.E(zqe.Invalid, "unsupported search format: %s", format)
	}
}

func handlePcapSearch(c *Core, w http.ResponseWriter, r *http.Request) {
	s := extractSpace(c, w, r)
	if s == nil {
		return
	}

	ctx, cancel, err := s.StartOp(r.Context())
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	defer cancel()

	var req api.PcapSearch
	if err := req.FromQuery(r.URL.Query()); err != nil {
		respondError(c, w, r, zqe.E(zqe.Invalid, err))
		return
	}
	pcapstore := s.PcapStore()
	if pcapstore.Empty() {
		respondError(c, w, r, zqe.E(zqe.NotFound, "no pcap in this space"))
		return
	}
	reader, err := pcapstore.NewSearch(ctx, req)
	if err == pcap.ErrNoPcapsFound {
		respondError(c, w, r, zqe.E(zqe.NotFound, err))
		return
	}
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	defer reader.Close()
	w.Header().Set("Content-Type", "application/vnd.tcpdump.pcap")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s.pcap", reader.ID()))
	_, err = ctxio.Copy(ctx, w, reader)
	if err != nil {
		c.requestLogger(r).Error("Error writing packet response", zap.Error(err))
	}
}

func handleSpaceList(c *Core, w http.ResponseWriter, r *http.Request) {
	spaces, err := c.spaces.List(r.Context())
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	respond(c, w, r, http.StatusOK, spaces)
}

func handleSpaceGet(c *Core, w http.ResponseWriter, r *http.Request) {
	s := extractSpace(c, w, r)
	if s == nil {
		return
	}

	ctx, cancel, err := s.StartOp(r.Context())
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	defer cancel()

	info, err := s.Info(ctx)
	if err != nil {
		respondError(c, w, r, err)
		return
	}

	respond(c, w, r, http.StatusOK, info)
}

func handleSpacePost(c *Core, w http.ResponseWriter, r *http.Request) {
	var req api.SpacePostRequest
	if !request(c, w, r, &req) {
		return
	}

	sp, err := c.spaces.Create(r.Context(), req)
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	info, err := sp.Info(r.Context())
	if err != nil {
		respondError(c, w, r, err)
		return
	}

	respond(c, w, r, http.StatusOK, info)
}

func handleSubspacePost(c *Core, w http.ResponseWriter, r *http.Request) {
	s := extractSpace(c, w, r)
	if s == nil {
		return
	}

	ctx, cancel, err := s.StartOp(r.Context())
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	defer cancel()

	var req api.SubspacePostRequest
	if !request(c, w, r, &req) {
		return
	}

	sp, err := c.spaces.CreateSubspace(ctx, s, req)
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	info, err := sp.Info(ctx)
	if err != nil {
		respondError(c, w, r, err)
		return
	}

	respond(c, w, r, http.StatusOK, info)
}

func handleSpacePut(c *Core, w http.ResponseWriter, r *http.Request) {
	s := extractSpace(c, w, r)
	if s == nil {
		return
	}

	_, cancel, err := s.StartOp(r.Context())
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	defer cancel()

	var req api.SpacePutRequest
	if !request(c, w, r, &req) {
		return
	}
	if err := c.spaces.UpdateSpace(s, req); err != nil {
		respondError(c, w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleSpaceDelete(c *Core, w http.ResponseWriter, r *http.Request) {
	v := mux.Vars(r)
	id, ok := v["space"]
	if !ok {
		respondError(c, w, r, zqe.E(zqe.Invalid, "no space id in path"))
		return
	}

	if err := c.spaces.Delete(r.Context(), api.SpaceID(id)); err != nil {
		respondError(c, w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handlePcapPost(c *Core, w http.ResponseWriter, r *http.Request) {
	if !c.HasZeek() {
		respondError(c, w, r, zqe.E(zqe.Invalid, "pcap post not supported: Zeek not found"))
		return
	}
	logger := c.requestLogger(r)

	s := extractSpace(c, w, r)
	if s == nil {
		return
	}

	ctx, cancel, err := s.StartOp(r.Context())
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	defer cancel()

	var req api.PcapPostRequest
	if !request(c, w, r, &req) {
		return
	}

	op, warnings, err := ingest.NewPcapOp(ctx, s, req.Path, c.suricata, c.zeek)
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/ndjson")
	w.WriteHeader(http.StatusAccepted)
	pipe := jsonpipe.New(w)
	taskID := c.nextTaskID()
	taskStart := api.TaskStart{Type: "TaskStart", TaskID: taskID}
	if err := pipe.Send(taskStart); err != nil {
		logger.Warn("Error sending payload", zap.Error(err))
		return
	}
	for _, warning := range warnings {
		err := pipe.Send(api.PcapPostWarning{
			Type:    "PcapPostWarning",
			Warning: warning,
		})
		if err != nil {
			logger.Warn("error sending payload", zap.Error(err))
			return
		}
	}
	ticker := time.NewTicker(time.Second * 2)
	defer ticker.Stop()
	for {
		var done bool
		select {
		case <-op.Done():
			done = true
		case <-op.Snap():
		case <-ticker.C:
		}

		sum, err := s.Storage().Summary(ctx)
		if err != nil {
			logger.Warn("Error reading storage summary", zap.Error(err))
			return
		}

		status := op.Status()
		status.Span = &sum.Span
		if err := pipe.Send(status); err != nil {
			logger.Warn("Error sending payload", zap.Error(err))
			return
		}
		if done {
			break
		}
	}
	taskEnd := api.TaskEnd{Type: "TaskEnd", TaskID: taskID}

	if ctx.Err() != nil {
		taskEnd.Error = &api.Error{Type: "Error", Message: "pcap post operation canceled"}
	} else if operr := op.Err(); operr != nil {
		var ok bool
		taskEnd.Error, ok = operr.(*api.Error)
		if !ok {
			taskEnd.Error = &api.Error{Type: "Error", Message: operr.Error()}
		}
	}
	if err := pipe.SendFinal(taskEnd); err != nil {
		logger.Warn("Error sending payload", zap.Error(err))
		return
	}
}

func handleLogPost(c *Core, w http.ResponseWriter, r *http.Request) {
	s := extractSpace(c, w, r)
	if s == nil {
		return
	}
	ctx, cancel, err := s.StartOp(r.Context())
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	defer cancel()

	var req api.LogPostRequest
	if !request(c, w, r, &req) {
		return
	}
	if len(req.Paths) == 0 {
		respondError(c, w, r, zqe.E(zqe.Invalid, "empty paths"))
		return
	}
	op, err := ingest.NewLogOp(ctx, s.Storage(), req)
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/ndjson")
	w.WriteHeader(http.StatusAccepted)
	logger := c.requestLogger(r)
	pipe := jsonpipe.New(w)
	if err := pipe.SendStart(0); err != nil {
		logger.Warn("error sending payload", zap.Error(err))
		return
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
loop:
	for {
		select {
		case warning, ok := <-op.Status():
			if !ok {
				break loop
			}
			err := pipe.Send(api.LogPostWarning{
				Type:    "LogPostWarning",
				Warning: warning,
			})
			if err != nil {
				logger.Warn("error sending payload", zap.Error(err))
				return
			}
		case <-ticker.C:
			if err := pipe.Send(op.Stats()); err != nil {
				logger.Warn("error sending payload", zap.Error(err))
				return
			}
		}
	}
	// send final status
	if err := pipe.Send(op.Stats()); err != nil {
		logger.Warn("error sending payload", zap.Error(err))
		return
	}
	if err := pipe.SendEnd(0, op.Error()); err != nil {
		logger.Warn("error sending payload", zap.Error(err))
		return
	}
}

func handleLogStream(c *Core, w http.ResponseWriter, r *http.Request) {
	s := extractSpace(c, w, r)
	if s == nil {
		return
	}
	ctx, cancel, err := s.StartOp(r.Context())
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	defer cancel()

	form, err := r.MultipartReader()
	if err != nil {
		respondError(c, w, r, zqe.ErrInvalid(err))
		return
	}

	zctx := resolver.NewContext()
	zr := ingest.NewMultipartLogReader(form, zctx)

	if r.URL.Query().Get("stop_err") != "" {
		zr.SetStopOnError()
	}

	if err := s.Storage().Write(ctx, zctx, zr); err != nil {
		respondError(c, w, r, err)
		return
	}

	respond(c, w, r, http.StatusOK, api.LogPostResponse{
		Type:      "LogPostResponse",
		BytesRead: zr.BytesRead(),
		Warnings:  zr.Warnings(),
	})
}

func handleIndexPost(c *Core, w http.ResponseWriter, r *http.Request) {
	s := extractSpace(c, w, r)
	if s == nil {
		return
	}
	ctx, cancel, err := s.StartOp(r.Context())
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	defer cancel()

	var req api.IndexPostRequest
	if !request(c, w, r, &req) {
		return
	}

	store, ok := s.Storage().(*archivestore.Storage)
	if !ok {
		respondError(c, w, r, zqe.E(zqe.Invalid, "space storage does not support creating indexes"))
		return
	}
	if err := store.IndexCreate(ctx, req); err != nil {
		respondError(c, w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleIndexSearch(c *Core, w http.ResponseWriter, r *http.Request) {
	s := extractSpace(c, w, r)
	if s == nil {
		return
	}
	ctx, cancel, err := s.StartOp(r.Context())
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	defer cancel()

	var req api.IndexSearchRequest
	if !request(c, w, r, &req) {
		return
	}

	store, ok := s.Storage().(search.IndexSearcher)
	if !ok {
		respondError(c, w, r, zqe.E(zqe.Invalid, "space storage does not support index search"))
		return
	}
	srch, err := search.NewIndexSearchOp(ctx, store, req)
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	defer srch.Close()

	out, err := getSearchOutput(w, r)
	if err != nil {
		respondError(c, w, r, err)
		return
	}

	w.Header().Set("Content-Type", out.ContentType())
	if err := srch.Run(out); err != nil {
		c.requestLogger(r).Warn("Error writing response", zap.Error(err))
	}
}

type ArchiveStater interface {
	ArchiveStat(context.Context, *resolver.Context) (zbuf.ReadCloser, error)
}

func handleArchiveStat(c *Core, w http.ResponseWriter, r *http.Request) {
	s := extractSpace(c, w, r)
	if s == nil {
		return
	}
	ctx, cancel, err := s.StartOp(r.Context())
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	defer cancel()

	store, ok := s.Storage().(ArchiveStater)
	if !ok {
		respondError(c, w, r, zqe.E(zqe.Invalid, "space storage does not support archive stat"))
		return
	}
	rc, err := store.ArchiveStat(ctx, resolver.NewContext())
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	defer rc.Close()

	out, err := getSearchOutput(w, r)
	if err != nil {
		respondError(c, w, r, err)
		return
	}

	w.Header().Set("Content-Type", out.ContentType())
	if err := search.SendFromReader(out, rc); err != nil {
		c.requestLogger(r).Warn("Error writing response", zap.Error(err))
	}
}

func extractSpace(c *Core, w http.ResponseWriter, r *http.Request) space.Space {
	v := mux.Vars(r)
	id, ok := v["space"]
	if !ok {
		respondError(c, w, r, zqe.E(zqe.Invalid, "no space id in path"))
		return nil
	}
	s, err := c.spaces.Get(api.SpaceID(id))
	if err != nil {
		respondError(c, w, r, err)
		return nil
	}
	return s
}

package zqd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/brimsec/zq/pcap"
	"github.com/brimsec/zq/pkg/ctxio"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/ingest"
	"github.com/brimsec/zq/zqd/search"
	"github.com/brimsec/zq/zqd/space"
	"github.com/brimsec/zq/zqe"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func errorResponse(e error) (status int, ae *api.Error) {
	status = http.StatusInternalServerError
	ae = &api.Error{}

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

	ae.Type = ze.Kind.String()
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
	respond(c, w, r, status, ae)
}

func request(c *Core, w http.ResponseWriter, r *http.Request, apiobj interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(apiobj); err != nil {
		respondError(c, w, r, zqe.E(zqe.Invalid, err))
		return false
	}
	return true
}

func handleSearch(c *Core, w http.ResponseWriter, r *http.Request) {
	var req api.SearchRequest
	if !request(c, w, r, &req) {
		return
	}

	s, err := space.Open(c.Root, req.Space)
	if err != nil {
		respondError(c, w, r, err)
		return
	}

	ctx, cancel, err := c.startSpaceOp(r.Context(), s.Name())
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	defer cancel()

	srch, err := search.NewSearch(ctx, s, req)
	if err != nil {
		// XXX This always returns bad request but should return status codes
		// that reflect the nature of the returned error.
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer srch.Close()

	var out search.Output
	format := r.URL.Query().Get("format")
	switch format {
	case "zjson", "json":
		// XXX Should write appropriate ndjson content header.
		out = search.NewJSONOutput(w, search.DefaultMTU)
	case "bzng":
		// XXX Should write appropriate bzng content header.
		out = search.NewBzngOutput(w)
	default:
		respondError(c, w, r, zqe.E(zqe.Invalid, "unsupported format: %s", format))
		return
	}
	// XXX This always returns bad request but should return status codes
	// that reflect the nature of the returned error.
	w.Header().Set("Content-Type", "application/ndjson")
	if err = srch.Run(out); err != nil {
		c.requestLogger(r).Warn("Error writing response", zap.Error(err))
	}
}

func handlePacketSearch(c *Core, w http.ResponseWriter, r *http.Request) {
	s := extractSpace(c, w, r)
	if s == nil {
		return
	}

	ctx, cancel, err := c.startSpaceOp(r.Context(), s.Name())
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	defer cancel()

	var req api.PacketSearch
	if err := req.FromQuery(r.URL.Query()); err != nil {
		respondError(c, w, r, zqe.E(zqe.Invalid, err))
		return
	}
	reader, err := s.PcapSearch(ctx, req)
	if err == pcap.ErrNoPacketsFound {
		respondError(c, w, r, zqe.E(zqe.NotFound, err))
		return
	}
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	defer reader.Close()
	w.WriteHeader(http.StatusAccepted)
	w.Header().Set("Content-Type", "application/vnd.tcpdump.pcap")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%s.pcap", reader.ID()))
	_, err = ctxio.Copy(ctx, w, reader)
	if err != nil {
		c.requestLogger(r).Error("Error writing packet response", zap.Error(err))
	}
}

func handleSpaceList(c *Core, w http.ResponseWriter, r *http.Request) {
	info, err := ioutil.ReadDir(c.Root)
	if err != nil {
		respondError(c, w, r, err)
		return
	}

	spaces := []string{}
	for _, subdir := range info {
		if !subdir.IsDir() {
			continue
		}
		s, err := space.Open(c.Root, subdir.Name())
		if err != nil {
			continue
		}
		spaces = append(spaces, s.Name())
	}

	respond(c, w, r, http.StatusOK, spaces)
}

func handleSpaceGet(c *Core, w http.ResponseWriter, r *http.Request) {
	s := extractSpace(c, w, r)
	if s == nil {
		return
	}

	_, cancel, err := c.startSpaceOp(r.Context(), s.Name())
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	defer cancel()

	info, err := s.Info()
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

	_, cancel, err := c.startSpaceOp(r.Context(), req.Name)
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	defer cancel()

	s, err := space.Create(c.Root, req.Name, req.DataDir)
	if err != nil {
		respondError(c, w, r, err)
		return
	}

	res := api.SpacePostResponse{
		Name:    s.Name(),
		DataDir: s.DataPath(),
	}
	respond(c, w, r, http.StatusOK, res)
}

func handleSpaceDelete(c *Core, w http.ResponseWriter, r *http.Request) {
	s := extractSpace(c, w, r)
	if s == nil {
		return
	}
	cancel, ok := c.haltSpaceOpsForDelete(s.Name())
	if !ok {
		respondError(c, w, r, zqe.E(zqe.Conflict))
		return
	}
	defer cancel()

	if err := s.Delete(); err != nil {
		respondError(c, w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handlePacketPost(c *Core, w http.ResponseWriter, r *http.Request) {
	if !c.HasZeek() {
		respondError(c, w, r, zqe.E(zqe.Invalid, "packet post not supported: zeek not found"))
		return
	}
	logger := c.requestLogger(r)

	s := extractSpace(c, w, r)
	if s == nil {
		return
	}

	ctx := r.Context()
	ctx, cancel, err := c.startSpaceOp(r.Context(), s.Name())
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	defer cancel()

	var req api.PacketPostRequest
	if !request(c, w, r, &req) {
		return
	}

	proc, err := ingest.Pcap(ctx, s, req.Path, c.ZeekLauncher, c.SortLimit)
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/ndjson")
	w.WriteHeader(http.StatusAccepted)
	pipe := api.NewJSONPipe(w)
	taskID := c.getTaskID()
	taskStart := api.TaskStart{Type: "TaskStart", TaskID: taskID}
	if err = pipe.Send(taskStart); err != nil {
		logger.Warn("Error sending payload", zap.Error(err))
		return
	}
	ticker := time.NewTicker(time.Second * 2)
	defer ticker.Stop()
	for {
		var done bool
		select {
		case <-proc.Done():
			done = true
		case <-proc.Snap():
		case <-ticker.C:
		}

		var minTs, maxTs *nano.Ts
		if minTs, maxTs, err = s.GetTimes(); err != nil {
			break
		}
		status := api.PacketPostStatus{
			Type:           "PacketPostStatus",
			StartTime:      proc.StartTime,
			UpdateTime:     nano.Now(),
			PacketSize:     proc.PcapSize,
			PacketReadSize: proc.PcapReadSize(),
			SnapshotCount:  proc.SnapshotCount(),
			MinTime:        minTs,
			MaxTime:        maxTs,
		}
		if err := pipe.Send(status); err != nil {
			logger.Warn("Error sending payload", zap.Error(err))
			return
		}
		if done {
			break
		}
	}
	taskEnd := api.TaskEnd{Type: "TaskEnd", TaskID: taskID}
	if err := proc.Err(); err != nil {
		var ok bool
		taskEnd.Error, ok = err.(*api.Error)
		if !ok {
			taskEnd.Error = &api.Error{Type: "Error", Message: err.Error()}
		}
	}
	if err = pipe.SendFinal(taskEnd); err != nil {
		logger.Warn("Error sending payload", zap.Error(err))
		return
	}
}

func handleLogPost(c *Core, w http.ResponseWriter, r *http.Request) {
	s := extractSpace(c, w, r)
	if s == nil {
		return
	}

	ctx, cancel, err := c.startSpaceOp(r.Context(), s.Name())
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
	w.Header().Set("Content-Type", "application/ndjson")
	w.WriteHeader(http.StatusAccepted)

	pipe := api.NewJSONPipe(w)
	err = ingest.Logs(ctx, pipe, s, req.Paths, req.JSONTypeConfig, c.SortLimit)
	if err != nil {
		c.requestLogger(r).Warn("Error during log ingest", zap.Error(err))
	}
}

func extractSpace(c *Core, w http.ResponseWriter, r *http.Request) *space.Space {
	v := mux.Vars(r)
	name, ok := v["space"]
	if !ok {
		respondError(c, w, r, zqe.E(zqe.Invalid, "no space name in path"))
		return nil
	}
	s, err := space.Open(c.Root, name)
	if err != nil {
		respondError(c, w, r, err)
		return nil
	}
	return s
}

package service

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/service/auth"
	"github.com/brimdata/zed/service/jsonpipe"
	"github.com/brimdata/zed/service/search"
	"github.com/brimdata/zed/zqe"
	"github.com/brimdata/zed/zson"
	"github.com/gorilla/mux"
	"github.com/segmentio/ksuid"
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
	case zqe.NoCredentials:
		status = http.StatusUnauthorized
	case zqe.Forbidden:
		status = http.StatusForbidden
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
	proc, err := compiler.ParseProc(req.ZQL)
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
	pool, err := c.root.OpenPool(r.Context(), req.Pool)
	if err != nil {
		if errors.Is(err, lake.ErrPoolNotFound) {
			err = zqe.ErrNotFound(err)
		}
		respondError(c, w, r, err)
		return
	}
	srch, err := search.NewSearchOp(req, c.requestLogger(r))
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	out, err := getSearchOutput(w, r)
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	w.Header().Set("Content-Type", out.ContentType())
	if err := srch.Run(r.Context(), pool, out, 0); err != nil {
		c.requestLogger(r).Warn("Error writing response", zap.Error(err))
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

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

func handlePoolList(c *Core, w http.ResponseWriter, r *http.Request) {
	var pools []api.Pool
	// XXX (nibs) This should run ScanPools but needs to be serialized as a json
	// array of pool. Revisit.
	for _, pool := range c.root.ListPools() {
		pools = append(pools, api.Pool{
			ID:   pool.ID,
			Name: pool.Name,
		})
	}
	respond(c, w, r, http.StatusOK, pools)
}

func handlePoolGet(c *Core, w http.ResponseWriter, r *http.Request) {
	id, ok := extractPoolID(c, w, r)
	if !ok {
		return
	}
	pool, err := c.root.OpenPool(r.Context(), id)
	if errors.Is(err, lake.ErrPoolNotFound) {
		respondError(c, w, r, zqe.ErrNotFound("pool %q not found", id))
		return
	}
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	snap, err := pool.Log().Head(r.Context())
	if errors.Is(err, journal.ErrEmpty) {
		respond(c, w, r, http.StatusOK, api.Pool{ID: pool.ID, Name: pool.Name})
		return
	}
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	info, err := pool.Info(r.Context(), snap)
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	respond(c, w, r, http.StatusOK, api.PoolInfo{
		Pool: api.Pool{ID: pool.ID, Name: pool.Name},
		Size: info.Size,
		Span: info.Span,
	})
}

func handlePoolPost(c *Core, w http.ResponseWriter, r *http.Request) {
	var req api.PoolPostRequest
	if !request(c, w, r, &req) {
		return
	}
	pool, err := c.root.CreatePool(r.Context(), req.Name, req.Keys, req.Order, req.Thresh)
	if err != nil {
		if errors.Is(err, lake.ErrPoolExists) {
			err = zqe.ErrConflict(err)
		}
		respondError(c, w, r, err)
		return
	}
	respond(c, w, r, http.StatusOK, api.Pool{Name: pool.Name, ID: pool.ID})
}

func handlePoolPut(c *Core, w http.ResponseWriter, r *http.Request) {
	var req api.PoolPutRequest
	if !request(c, w, r, &req) {
		return
	}
	id, ok := extractPoolID(c, w, r)
	if !ok {
		return
	}

	if err := c.root.RenamePool(r.Context(), id, req.Name); err != nil {
		if errors.Is(err, lake.ErrPoolExists) {
			err = zqe.ErrConflict(err)
		} else if errors.Is(err, lake.ErrPoolNotFound) {
			err = zqe.ErrNotFound(err)
		}
		respondError(c, w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handlePoolDelete(c *Core, w http.ResponseWriter, r *http.Request) {
	id, ok := extractPoolID(c, w, r)
	if !ok {
		return
	}
	if err := c.root.RemovePool(r.Context(), id); err != nil {
		respondError(c, w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleLogPost(c *Core, w http.ResponseWriter, r *http.Request) {
	var req api.LogPostRequest
	if !request(c, w, r, &req) {
		return
	}
	if len(req.Paths) == 0 {
		respondError(c, w, r, zqe.E(zqe.Invalid, "empty paths"))
		return
	}
	id, ok := extractPoolID(c, w, r)
	if !ok {
		return
	}

	pool, err := c.root.OpenPool(r.Context(), id)
	if err != nil {
		respondError(c, w, r, err)
		return
	}

	op, err := NewLogOp(r.Context(), pool, req)
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
	err = op.Error()
	if err == nil {
		// XXX Figure out some mechanism for propagating user and message.
		err = pool.Commit(r.Context(), op.Commit(), nano.Now(), "", "")
	}
	if err == nil {
		res := api.LogPostResponse{Type: "LogPostResponse", Commit: op.Commit()}
		if err := pipe.Send(res); err != nil {
			logger.Warn("error sending payload", zap.Error(err))
			return
		}
	}
	if err := pipe.SendEnd(0, err); err != nil {
		logger.Warn("error sending payload", zap.Error(err))
		return
	}
}

func handleLogStream(c *Core, w http.ResponseWriter, r *http.Request) {
	form, err := r.MultipartReader()
	if err != nil {
		respondError(c, w, r, zqe.ErrInvalid(err))
		return
	}
	id, ok := extractPoolID(c, w, r)
	if !ok {
		return
	}
	pool, err := c.root.OpenPool(r.Context(), id)
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	zctx := zson.NewContext()
	zr := NewMultipartLogReader(form, zctx)
	if r.URL.Query().Get("stop_err") != "" {
		zr.SetStopOnError()
	}
	commit, err := pool.Add(r.Context(), zr)
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	// XXX Figure out some mechanism for propagating user and message.
	if err := pool.Commit(r.Context(), commit, nano.Now(), "", ""); err != nil {
		respondError(c, w, r, err)
		return
	}

	// XXX add a separate commit hook
	respond(c, w, r, http.StatusOK, api.LogPostResponse{
		Type:      "LogPostResponse",
		BytesRead: zr.BytesRead(),
		Commit:    commit,
		Warnings:  zr.Warnings(),
	})
}

/* XXX Not yet
func handleIndexPost(c *Core, w http.ResponseWriter, r *http.Request) {
	var req api.IndexPostRequest
	if !request(c, w, r, &req) {
		return
	}
	// func (r *Root) AddIndex(ctx context.Context, indices []index.Index) error {
	if err := c.root.AddIndex(r.Context(), req); err != nil {
		respondError(c, w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleIndexSearch(c *Core, w http.ResponseWriter, r *http.Request) {
	var req api.IndexSearchRequest
	if !request(c, w, r, &req) {
		return
	}
	id, ok := extractPoolID(c, w, r)
	if !ok {
		return
	}
	store, err := c.mgr.GetStorage(r.Context(), id)
	if err != nil {
		respondError(c, w, r, err)
		return
	}
	as, ok := store.(*archivestore.Storage)
	if !ok {
		respondError(c, w, r, zqe.E(zqe.Invalid, "pool storage does not support index search"))
		return
	}

	srch, err := search.NewIndexSearchOp(r.Context(), as, req)
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
*/

func extractPoolID(c *Core, w http.ResponseWriter, r *http.Request) (ksuid.KSUID, bool) {
	v := mux.Vars(r)
	s, ok := v["pool"]
	if !ok {
		respondError(c, w, r, zqe.ErrInvalid("no pool id in path"))
		return ksuid.Nil, false
	}
	id, err := lake.ParseID(s)
	if err != nil {
		respondError(c, w, r, zqe.ErrInvalid(err))
		return ksuid.Nil, false
	}
	return id, true
}

func handleAuthIdentityGet(c *Core, w http.ResponseWriter, r *http.Request) {
	ident := auth.IdentityFromContext(r.Context())
	respond(c, w, r, http.StatusOK, api.AuthIdentityResponse{
		TenantID: string(ident.TenantID),
		UserID:   string(ident.UserID),
	})
}

func handleAuthMethodGet(c *Core, w http.ResponseWriter, r *http.Request) {
	if c.auth == nil {
		respond(c, w, r, http.StatusOK, api.AuthMethodResponse{Kind: api.AuthMethodNone})
		return
	}
	respond(c, w, r, http.StatusOK, c.auth.MethodResponse())
}

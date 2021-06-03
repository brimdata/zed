package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/commit"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/service/auth"
	"github.com/brimdata/zed/service/jsonpipe"
	"github.com/brimdata/zed/service/search"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/anyio"
	"github.com/brimdata/zed/zio/jsonio"
	"github.com/brimdata/zed/zqe"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
)

func handleASTPost(c *Core, w *ResponseWriter, r *Request) {
	var req api.ASTRequest
	accept := r.Header.Get("Accept")
	if accept != api.MediaTypeJSON && !api.IsAmbiguousMediaType(accept) {
		w.Error(zqe.ErrInvalid("unsupported accept header: %s", w.ContentType()))
		return
	}
	if !r.Unmarshal(w, &req) {
		return
	}
	proc, err := compiler.ParseProc(req.ZQL)
	if err != nil {
		w.Error(err)
		return
	}
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(proc); err != nil {
		w.Error(err)
		return
	}
}

func handleSearch(c *Core, w *ResponseWriter, r *Request) {
	var req api.SearchRequest
	if !r.Unmarshal(w, &req) {
		return
	}
	pool, err := c.root.OpenPool(r.Context(), ksuid.KSUID(req.Pool))
	if err != nil {
		if errors.Is(err, lake.ErrPoolNotFound) {
			err = zqe.ErrNotFound(err)
		}
		w.Error(err)
		return
	}
	srch, err := search.NewSearchOp(req, r.Logger)
	if err != nil {
		w.Error(err)
		return
	}
	out, err := getSearchOutput(w.ResponseWriter, r)
	if err != nil {
		w.Error(err)
		return
	}
	w.Header().Set("Content-Type", out.ContentType())
	if err := srch.Run(r.Context(), c.root, pool, out, 0); err != nil {
		r.Logger.Warn("Error writing response", zap.Error(err))
	}
}

func getSearchOutput(w http.ResponseWriter, r *Request) (search.Output, error) {
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

func handlePoolList(c *Core, w *ResponseWriter, r *Request) {
	zw := w.ZioWriterWithOpts(anyio.WriterOpts{JSON: jsonio.WriterOpts{ForceArray: true}})
	if zw == nil {
		return
	}
	if err := c.root.ScanPools(r.Context(), zw); err != nil {
		r.Logger.Warn("Error scanning pools", zap.Error(err))
		return
	}
	zw.Close()
}

func handlePoolGet(c *Core, w *ResponseWriter, r *Request) {
	id, ok := r.PoolID(w)
	if !ok {
		return
	}
	pool, err := c.root.OpenPool(r.Context(), id)
	if errors.Is(err, lake.ErrPoolNotFound) {
		err = zqe.ErrNotFound("pool %q not found", id)
	}
	if err != nil {
		w.Error(err)
		return
	}
	w.Respond(http.StatusOK, pool.PoolConfig)
}

func handlePoolStats(c *Core, w *ResponseWriter, r *Request) {
	id, ok := r.PoolID(w)
	if !ok {
		return
	}
	pool, err := c.root.OpenPool(r.Context(), id)
	if errors.Is(err, lake.ErrPoolNotFound) {
		err = zqe.ErrNotFound("pool %q not found", id)
	}
	if err != nil {
		w.Error(err)
		return
	}
	snap, err := snapshotAt(r.Context(), pool, r.URL.Query().Get("at"))
	if err != nil {
		if errors.Is(err, journal.ErrEmpty) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Error(err)
		return
	}
	info, err := pool.Stats(r.Context(), snap)
	if err != nil {
		w.Error(err)
		return
	}
	w.Respond(http.StatusOK, info)
}

func handlePoolPost(c *Core, w *ResponseWriter, r *Request) {
	var req api.PoolPostRequest
	if !r.Unmarshal(w, &req) {
		return
	}
	pool, err := c.root.CreatePool(r.Context(), req.Name, req.Layout, req.Thresh)
	if err != nil {
		if errors.Is(err, lake.ErrPoolExists) {
			err = zqe.ErrConflict(err)
		}
		w.Error(err)
		return
	}
	w.Respond(http.StatusOK, pool.PoolConfig)
}

func handlePoolPut(c *Core, w *ResponseWriter, r *Request) {
	var req api.PoolPutRequest
	if !r.Unmarshal(w, &req) {
		return
	}
	id, ok := r.PoolID(w)
	if !ok {
		return
	}

	if err := c.root.RenamePool(r.Context(), id, req.Name); err != nil {
		if errors.Is(err, lake.ErrPoolExists) {
			err = zqe.ErrConflict(err)
		} else if errors.Is(err, lake.ErrPoolNotFound) {
			err = zqe.ErrNotFound(err)
		}
		w.Error(err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handlePoolDelete(c *Core, w *ResponseWriter, r *Request) {
	id, ok := r.PoolID(w)
	if !ok {
		return
	}
	if err := c.root.RemovePool(r.Context(), id); err != nil {
		w.Error(err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type warningCollector []string

func (w *warningCollector) Warn(msg string) error {
	*w = append(*w, msg)
	return nil
}

func handleAdd(c *Core, w *ResponseWriter, r *Request) {
	id, ok := r.PoolID(w)
	if !ok {
		return
	}
	pool, err := c.root.OpenPool(r.Context(), id)
	if err != nil {
		w.Error(err)
		return
	}
	zr, err := anyio.NewReader(r.Body, zson.NewContext())
	if err != nil {
		w.Error(err)
		return
	}
	warnings := warningCollector{}
	zr = zio.NewWarningReader(zr, &warnings)
	commit, err := pool.Add(r.Context(), zr)
	if err != nil {
		w.Error(err)
		return
	}
	w.Respond(http.StatusOK, api.AddResponse{
		Warnings: warnings,
		Commit:   commit,
	})
}

func handleCommit(c *Core, w *ResponseWriter, r *Request) {
	id, ok := r.PoolID(w)
	if !ok {
		return
	}
	pool, err := c.root.OpenPool(r.Context(), id)
	if err != nil {
		w.Error(err)
		return
	}
	var commit api.CommitRequest
	if !r.Unmarshal(w, &commit) {
		return
	}
	commitID, ok := r.TagFromPath("commit", w)
	if !ok {
		return
	}
	err = pool.Commit(r.Context(), commitID, commit.Date, commit.Author, commit.Message)
	if err != nil {
		w.Error(err)
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleScanStaging(c *Core, w *ResponseWriter, r *Request) {
	id, ok := r.PoolID(w)
	if !ok {
		return
	}
	pool, err := c.root.OpenPool(r.Context(), id)
	if err != nil {
		w.Error(err)
		return
	}
	var ids []ksuid.KSUID
	if tags := r.URL.Query()["tag"]; tags != nil {
		if ids, err = api.ParseKSUIDs(tags); err != nil {
			w.Error(zqe.ErrInvalid(err))
			return
		}
	}
	if len(ids) == 0 {
		ids, err = pool.ListStagedCommits(r.Context())
		if err != nil {
			w.Error(err)
			return
		}
		if len(ids) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	zw := w.ZioWriter()
	defer zw.Close()
	if err := pool.ScanStaging(r.Context(), zw, ids); err != nil {
		w.Error(err)
		return
	}
}

func handleScanSegments(c *Core, w *ResponseWriter, r *Request) {
	poolID, ok := r.PoolID(w)
	if !ok {
		return
	}
	pool, err := c.root.OpenPool(r.Context(), poolID)
	if err != nil {
		w.Error(err)
		return
	}
	snap, err := snapshotAt(r.Context(), pool, r.URL.Query().Get("at"))
	if err != nil {
		if errors.Is(err, journal.ErrEmpty) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Error(err)
		return
	}
	zw := w.ZioWriter()
	defer zw.Close()
	if r.URL.Query().Get("partition") == "T" {
		if err := pool.ScanPartitions(r.Context(), zw, snap, nil); err != nil {
			w.Error(err)
		}
		return
	}
	if err := pool.ScanSegments(r.Context(), zw, snap, nil); err != nil {
		w.Error(err)
		return
	}
}

func handleScanLog(c *Core, w *ResponseWriter, r *Request) {
	id, ok := r.PoolID(w)
	if !ok {
		return
	}
	pool, err := c.root.OpenPool(r.Context(), id)
	if err != nil {
		w.Error(err)
		return
	}
	zw := w.ZioWriter()
	defer zw.Close()
	// XXX Support head/tail references in api.
	zr, err := pool.Log().OpenAsZNG(r.Context(), 0, 0)
	if err != nil {
		w.Error(err)
		return
	}
	if err := zio.CopyWithContext(r.Context(), zw, zr); err != nil {
		w.Error(err)
	}
}

func handleLogPostPaths(c *Core, w *ResponseWriter, r *Request) {
	var req api.LogPostRequest
	if !r.Unmarshal(w, &req) {
		return
	}
	if len(req.Paths) == 0 {
		w.Error(zqe.E(zqe.Invalid, "empty paths"))
		return
	}
	id, ok := r.PoolID(w)
	if !ok {
		return
	}
	pool, err := c.root.OpenPool(r.Context(), id)
	if err != nil {
		w.Error(err)
		return
	}
	op, err := NewLogOp(r.Context(), pool, req)
	if err != nil {
		w.Error(err)
		return
	}
	if w.ContentType() == api.MediaTypeJSON {
		w.SetContentType(api.MediaTypeNDJSON)
	}
	w.WriteHeader(http.StatusAccepted)
	pipe := jsonpipe.New(w.ResponseWriter)
	if err := pipe.SendStart(0); err != nil {
		w.Logger.Warn("error sending payload", zap.Error(err))
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
				w.Logger.Warn("error sending payload", zap.Error(err))
				return
			}
		case <-ticker.C:
			if err := pipe.Send(op.Stats()); err != nil {
				w.Logger.Warn("error sending payload", zap.Error(err))
				return
			}
		}
	}
	// send final status
	if err := pipe.Send(op.Stats()); err != nil {
		w.Logger.Warn("error sending payload", zap.Error(err))
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
			w.Logger.Warn("error sending payload", zap.Error(err))
			return
		}
	}
	if err := pipe.SendEnd(0, err); err != nil {
		w.Logger.Warn("error sending payload", zap.Error(err))
		return
	}
}

// Deprecated. Use handlePoolAdd instead.
func handleLogPost(c *Core, w *ResponseWriter, r *Request) {
	form, err := r.MultipartReader()
	if err != nil {
		w.Error(zqe.ErrInvalid(err))
		return
	}
	id, ok := r.PoolID(w)
	if !ok {
		return
	}
	pool, err := c.root.OpenPool(r.Context(), id)
	if err != nil {
		w.Error(err)
		return
	}
	zctx := zson.NewContext()
	zr := NewMultipartLogReader(form, zctx)
	if r.URL.Query().Get("stop_err") != "" {
		zr.SetStopOnError()
	}
	commit, err := pool.Add(r.Context(), zr)
	if err != nil {
		w.Error(err)
		return
	}
	// XXX Figure out some mechanism for propagating user and message.
	if err := pool.Commit(r.Context(), commit, nano.Now(), "", ""); err != nil {
		w.Error(err)
		return
	}

	// XXX add a separate commit hook
	w.Respond(http.StatusOK, api.LogPostResponse{
		Type:      "LogPostResponse",
		BytesRead: zr.BytesRead(),
		Commit:    commit,
		Warnings:  zr.Warnings(),
	})
}

/* XXX Not yet
func handleIndexPost(c *Core, w *Response, r *Request) {
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

func handleIndexSearch(c *Core, w *Response, r *Request) {
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

func handleAuthIdentityGet(c *Core, w *ResponseWriter, r *Request) {
	ident := auth.IdentityFromContext(r.Context())
	w.Respond(http.StatusOK, api.AuthIdentityResponse{
		TenantID: string(ident.TenantID),
		UserID:   string(ident.UserID),
	})
}

func handleAuthMethodGet(c *Core, w *ResponseWriter, r *Request) {
	if c.auth == nil {
		w.Respond(http.StatusOK, api.AuthMethodResponse{Kind: api.AuthMethodNone})
		return
	}
	w.Respond(http.StatusOK, c.auth.MethodResponse())
}

func snapshotAt(ctx context.Context, pool *lake.Pool, at string) (*commit.Snapshot, error) {
	var id journal.ID
	if at != "" {
		var err error
		id, err = parseJournalID(ctx, pool, at)
		if err != nil {
			return nil, err
		}
	}
	return pool.Log().Snapshot(ctx, id)
}

func parseJournalID(ctx context.Context, pool *lake.Pool, at string) (journal.ID, error) {
	if num, err := strconv.Atoi(at); err == nil {
		ok, err := pool.IsJournalID(ctx, journal.ID(num))
		if err != nil {
			return journal.Nil, err
		}
		if ok {
			return journal.ID(num), nil
		}
	}
	commitID, err := api.ParseKSUID(at)
	if err != nil {
		return journal.Nil, zqe.ErrInvalid("not a valid journal number or a commit tag: %s", at)
	}
	id, err := pool.Log().JournalIDOfCommit(ctx, 0, commitID)
	if err != nil {
		return journal.Nil, zqe.ErrInvalid("not a valid journal number or a commit tag: %s", at)
	}
	return id, nil
}

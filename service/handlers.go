package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/api/queryio"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/commit"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/service/auth"
	"github.com/brimdata/zed/service/search"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/anyio"
	"github.com/brimdata/zed/zio/jsonio"
	"github.com/brimdata/zed/zqe"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
	"go.uber.org/zap"
)

func handleQuery(c *Core, w *ResponseWriter, r *Request) {
	var req api.QueryRequest
	if !r.Unmarshal(w, &req) {
		return
	}
	query, err := compiler.ParseProc(req.Query)
	if err != nil {
		w.Error(zqe.ErrInvalid(err))
		return
	}
	noctrl, ok := r.BoolFromQuery("noctrl", w)
	if !ok {
		return
	}
	format, err := api.MediaTypeToFormat(r.Header.Get("Accept"))
	if err != nil {
		if !errors.Is(err, api.ErrMediaTypeUnspecified) {
			w.Error(zqe.ErrInvalid(err))
			return
		}
		format = "zjson"
	}
	d, err := queryio.NewDriver(zio.NopCloser(w), format, !noctrl)
	if err != nil {
		w.Error(err)
	}
	defer d.Close()
	err = driver.RunWithLakeAndStats(r.Context(), d, query, zson.NewContext(), c.root, nil, r.Logger, 0)
	if err != nil && !errors.Is(err, journal.ErrEmpty) {
		d.Error(err)
	}
}

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
	c.publishEvent("pool-new", api.EventPool{
		PoolID: pool.ID.String(),
	})
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
	c.publishEvent("pool-update", api.EventPool{
		PoolID: id.String(),
	})
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
	c.publishEvent("pool-delete", api.EventPool{
		PoolID: id.String(),
	})
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
	zr, err := anyio.NewReader(anyio.GzipReader(r.Body), zson.NewContext())
	if err != nil {
		w.Error(err)
		return
	}
	warnings := warningCollector{}
	zr = zio.NewWarningReader(zr, &warnings)
	kommit, err := pool.Add(r.Context(), zr)
	if err != nil {
		if errors.Is(err, commit.ErrEmptyTransaction) {
			err = zqe.ErrInvalid("no records in request")
		}
		w.Error(err)
		return
	}
	w.Respond(http.StatusOK, api.AddResponse{
		Warnings: warnings,
		Commit:   kommit,
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
	c.publishEvent("pool-commit", api.EventPoolCommit{
		CommitID: commitID.String(),
		PoolID:   pool.ID.String(),
	})
	w.WriteHeader(http.StatusNoContent)
}

func handleDelete(c *Core, w *ResponseWriter, r *Request) {
	id, ok := r.PoolID(w)
	if !ok {
		return
	}
	var args []string
	if !r.Unmarshal(w, &args) {
		return
	}
	tags, err := api.ParseKSUIDs(args)
	if err != nil {
		w.Error(zqe.ErrInvalid(err))
	}
	pool, err := c.root.OpenPool(r.Context(), id)
	if err != nil {
		w.Error(err)
		return
	}
	ids, err := pool.LookupTags(r.Context(), tags)
	if err != nil {
		w.Error(err)
		return
	}
	commit, err := pool.Delete(r.Context(), ids)
	if err != nil {
		w.Error(err)
		return
	}
	w.Marshal(api.StagedCommit{commit})
}

func handleSquash(c *Core, w *ResponseWriter, r *Request) {
	id, ok := r.PoolID(w)
	if !ok {
		return
	}
	pool, err := c.root.OpenPool(r.Context(), id)
	if err != nil {
		w.Error(err)
		return
	}
	var req api.SquashRequest
	if !r.Unmarshal(w, &req) {
		return
	}
	commit, err := pool.Squash(r.Context(), req.Commits)
	if err != nil {
		w.Error(err)
	}
	w.Marshal(api.StagedCommit{commit})
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
	zw := w.ZioWriter()
	defer zw.Close()
	if err := pool.ScanStaging(r.Context(), zw, ids); err != nil {
		if errors.Is(err, lake.ErrStagingEmpty) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
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

func handleEvents(c *Core, w *ResponseWriter, r *Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	subscription := make(chan []byte)
	c.subscriptionsMu.Lock()
	c.subscriptions[subscription] = struct{}{}
	c.subscriptionsMu.Unlock()
	for {
		select {
		case msg := <-subscription:
			if _, err := w.Write(msg); err != nil {
				w.Error(err)
				continue
			}
			if f, ok := w.ResponseWriter.(http.Flusher); ok {
				f.Flush()
			}
		case <-r.Context().Done():
			c.subscriptionsMu.Lock()
			delete(c.subscriptions, subscription)
			c.subscriptionsMu.Unlock()
			return
		}
	}
}

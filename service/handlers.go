package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/api/queryio"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/commit"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/service/auth"
	"github.com/brimdata/zed/service/search"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/anyio"
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
	branch, err := pool.OpenBranchByName(r.Context(), "main")
	if err != nil {
		if errors.Is(err, lake.ErrBranchNotFound) {
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
	if err := srch.Run(r.Context(), c.root, branch, out, 0); err != nil {
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

func handleIDs(c *Core, w *ResponseWriter, r *Request) {
	var req api.IDsRequest
	if !r.Unmarshal(w, &req) {
		return
	}
	poolID, branchID, err := c.root.IDs(r.Context(), req.Pool, req.Branch)
	if err != nil {
		w.Error(err)
		return
	}
	w.Respond(http.StatusOK, api.IDsResponse{poolID, branchID})
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
	//XXX app uses this for key range... should handle this differently
	// at minimum on a per-branch basis and app needs to be branch aware
	// If branch not specified, API endpoints here should just assume "main".
	branch, err := pool.OpenBranchByName(r.Context(), "main")
	if err != nil {
		w.Error(err)
		return
	}
	snap, err := branch.Log().Snapshot(r.Context(), 0)
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
	meta, err := pool.Main(r.Context())
	if err != nil {
		w.Error(err)
		return
	}
	c.publishEvent("pool-new", api.EventPool{
		PoolID: pool.ID.String(),
	})
	w.Respond(http.StatusOK, meta)
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

func handleBranchPost(c *Core, w *ResponseWriter, r *Request) {
	var req api.BranchPostRequest
	if !r.Unmarshal(w, &req) {
		return
	}
	poolID, ok := r.PoolID(w)
	if !ok {
		return
	}
	var parent, atTag ksuid.KSUID
	if req.ParentID != "" {
		var err error
		parent, err = api.ParseKSUID(req.ParentID)
		if err != nil {
			w.Error(zqe.ErrInvalid("invalid parent ID: %s", req.ParentID))
			return
		}
	}
	if req.At != "" {
		var err error
		atTag, err = api.ParseKSUID(req.At)
		if err != nil {
			w.Error(zqe.ErrInvalid("invalid parent at tag: %s", req.At))
			return
		}
	}
	branchRef, err := c.root.CreateBranch(r.Context(), poolID, req.Name, parent, atTag)
	if err != nil {
		if errors.Is(err, lake.ErrBranchExists) {
			err = zqe.ErrConflict(err)
		} else if errors.Is(err, lake.ErrPoolNotFound) {
			err = zqe.ErrNotFound(err)
		}
		w.Error(err)
		return
	}
	c.publishEvent("branch-update", api.EventBranch{
		PoolID:   poolID.String(),
		BranchID: branchRef.ID.String(),
	})
	w.Respond(http.StatusOK, branchRef)
}

func handleBranchMerge(c *Core, w *ResponseWriter, r *Request) {
	var req api.BranchMergeRequest
	if !r.Unmarshal(w, &req) {
		return
	}
	poolID, ok := r.PoolID(w)
	if !ok {
		return
	}
	branchID, ok := r.BranchID(w)
	if !ok {
		return
	}
	var at ksuid.KSUID
	if req.At != "" {
		var err error
		at, err = ksuid.Parse(req.At)
		if err != nil {
			w.Error(fmt.Errorf("merge endpoint: bad at option: %w", err))
			return
		}
	}
	commit, err := c.root.MergeBranch(r.Context(), poolID, branchID, at)
	if err != nil {
		w.Error(err)
		return
	}
	c.publishEvent("branch-merge", api.EventBranchCommit{
		CommitID: commit.String(),
		PoolID:   poolID.String(),
		BranchID: branchID.String(),
	})
	w.Respond(http.StatusOK, api.CommitResponse{Commit: commit})
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

func handleBranchDelete(c *Core, w *ResponseWriter, r *Request) {
	poolID, ok := r.PoolID(w)
	if !ok {
		return
	}
	branchID, ok := r.BranchID(w)
	if !ok {
		return
	}
	if err := c.root.RemoveBranch(r.Context(), poolID, branchID); err != nil {
		w.Error(err)
		return
	}
	c.publishEvent("branch-delete", api.EventPool{
		PoolID: poolID.String(),
		//XXX TBD BranchID: branchID.String(),
	})
	w.WriteHeader(http.StatusNoContent)
}

type warningCollector []string

func (w *warningCollector) Warn(msg string) error {
	*w = append(*w, msg)
	return nil
}

func handleBranchLoad(c *Core, w *ResponseWriter, r *Request) {
	poolID, ok := r.PoolID(w)
	if !ok {
		return
	}
	branchID, ok := r.BranchID(w)
	if !ok {
		return
	}
	pool, err := c.root.OpenPool(r.Context(), poolID)
	if err != nil {
		w.Error(err)
		return
	}
	branch, err := pool.OpenBranchByID(r.Context(), branchID)
	if err != nil {
		w.Error(err)
		return
	}
	commitJSON := r.Header.Get("Zed-Commit")
	var info api.CommitRequest
	if commitJSON != "" {
		if err := json.Unmarshal([]byte(commitJSON), &info); err != nil {
			w.Error(fmt.Errorf("load endpoint encountered invalid JSON in Zed-Commit header: %w", err))
			return
		}
	}
	if info.Date == 0 {
		info.Date = nano.Now()
	}
	zr, err := anyio.NewReader(anyio.GzipReader(r.Body), zson.NewContext())
	if err != nil {
		w.Error(err)
		return
	}
	warnings := warningCollector{}
	zr = zio.NewWarningReader(zr, &warnings)
	kommit, err := branch.Load(r.Context(), zr, info.Date, info.Author, info.Message)
	if err != nil {
		if errors.Is(err, commit.ErrEmptyTransaction) {
			err = zqe.ErrInvalid("no records in request")
		}
		w.Error(err)
		return
	}
	w.Respond(http.StatusOK, api.CommitResponse{
		Warnings: warnings,
		Commit:   kommit,
	})
	c.publishEvent("branch-commit", api.EventBranchCommit{
		CommitID: kommit.String(),
		PoolID:   pool.ID.String(),
		BranchID: branch.ID.String(),
	})
}

func handleDelete(c *Core, w *ResponseWriter, r *Request) {
	poolID, ok := r.PoolID(w)
	if !ok {
		return
	}
	branchID, ok := r.BranchID(w)
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
	pool, err := c.root.OpenPool(r.Context(), poolID)
	if err != nil {
		w.Error(err)
		return
	}
	branch, err := pool.OpenBranchByID(r.Context(), branchID)
	if err != nil {
		w.Error(err)
		return
	}
	ids, err := branch.LookupTags(r.Context(), tags)
	if err != nil {
		w.Error(err)
		return
	}
	commit, err := branch.Delete(r.Context(), ids)
	if err != nil {
		w.Error(err)
		return
	}
	w.Marshal(api.CommitResponse{Commit: commit})
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

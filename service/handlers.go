package service

import (
	"errors"
	"net/http"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/api/queryio"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/driver"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lake/branches"
	"github.com/brimdata/zed/lake/commits"
	"github.com/brimdata/zed/lake/index"
	"github.com/brimdata/zed/lake/journal"
	"github.com/brimdata/zed/lake/pools"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/service/auth"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/anyio"
	"github.com/brimdata/zed/zqe"
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
	accept := r.Header.Get("Accept")
	if accept == "" || accept == api.MediaTypeAny {
		accept = api.MediaTypeZSON
	}
	format, err := api.MediaTypeToFormat(accept)
	if err != nil && !errors.Is(err, api.ErrMediaTypeUnspecified) {
		w.Error(zqe.ErrInvalid(err))
		return
	}
	d, err := queryio.NewDriver(zio.NopCloser(w), format, !noctrl)
	if err != nil {
		w.Error(err)
	}
	defer d.Close()
	err = driver.RunWithLakeAndStats(r.Context(), d, query, zed.NewContext(), c.root, &req.Head, nil, r.Logger, 0)
	if err != nil && !errors.Is(err, journal.ErrEmpty) {
		d.Error(err)
	}
}

func handleBranchGet(c *Core, w *ResponseWriter, r *Request) {
	id, ok := r.PoolID(w, c.root)
	if !ok {
		return
	}
	branchName, ok := r.StringFromPath(w, "branch")
	if !ok {
		return
	}
	pool, err := c.root.OpenPool(r.Context(), id)
	if errors.Is(err, pools.ErrNotFound) {
		err = zqe.ErrNotFound("pool %q not found", id)
	}
	if err != nil {
		w.Error(err)
		return
	}
	if branchName != "" {
		branch, err := pool.LookupBranchByName(r.Context(), branchName)
		if err != nil {
			if errors.Is(err, branches.ErrNotFound) {
				err = zqe.ErrNotFound("branch %q not found", branchName)
			}
			w.Error(err)
			return
		}
		w.Respond(http.StatusOK, api.CommitResponse{Commit: branch.Commit})
		return
	}
	w.Respond(http.StatusOK, pool.Config)
}

func handlePoolStats(c *Core, w *ResponseWriter, r *Request) {
	id, ok := r.PoolID(w, c.root)
	if !ok {
		return
	}
	pool, err := c.root.OpenPool(r.Context(), id)
	if errors.Is(err, pools.ErrNotFound) {
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
	snap, err := branch.Pool().Snapshot(r.Context(), branch.Commit)
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
		if errors.Is(err, pools.ErrExists) {
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
	id, ok := r.PoolID(w, c.root)
	if !ok {
		return
	}
	if err := c.root.RenamePool(r.Context(), id, req.Name); err != nil {
		if errors.Is(err, pools.ErrExists) {
			err = zqe.ErrConflict(err)
		} else if errors.Is(err, pools.ErrNotFound) {
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
	poolID, ok := r.PoolID(w, c.root)
	if !ok {
		return
	}
	commit, err := lakeparse.ParseID(req.Commit)
	if err != nil {
		w.Error(zqe.ErrInvalid("invalid commit object: %s", req.Commit))
		return
	}
	branchRef, err := c.root.CreateBranch(r.Context(), poolID, req.Name, commit)
	if err != nil {
		if errors.Is(err, branches.ErrExists) {
			err = zqe.ErrConflict(err)
		} else if errors.Is(err, pools.ErrNotFound) {
			err = zqe.ErrNotFound(err)
		}
		w.Error(err)
		return
	}
	c.publishEvent("branch-update", api.EventBranch{
		PoolID: poolID.String(),
		Branch: branchRef.Name,
	})
	w.Respond(http.StatusOK, branchRef)
}

func handleRevertPost(c *Core, w *ResponseWriter, r *Request) {
	poolID, ok := r.PoolID(w, c.root)
	if !ok {
		return
	}
	branch, ok := r.StringFromPath(w, "branch")
	if !ok {
		return
	}
	commit, ok := r.CommitID(w)
	if !ok {
		return
	}
	message, ok := r.decodeCommitMessage(w)
	if !ok {
		return
	}
	commit, err := c.root.Revert(r.Context(), poolID, branch, commit, message.Author, message.Body)
	if err != nil {
		w.Error(err)
		return
	}
	c.publishEvent("branch-revert", api.EventBranchCommit{
		CommitID: commit.String(),
		PoolID:   poolID.String(),
		Branch:   branch,
	})
	w.Respond(http.StatusOK, api.CommitResponse{Commit: commit})
}

func handleBranchMerge(c *Core, w *ResponseWriter, r *Request) {
	poolID, ok := r.PoolID(w, c.root)
	if !ok {
		return
	}
	parentBranch, ok := r.StringFromPath(w, "branch")
	if !ok {
		return
	}
	childBranch, ok := r.StringFromPath(w, "child")
	if !ok {
		return
	}
	message, ok := r.decodeCommitMessage(w)
	if !ok {
		return
	}
	commit, err := c.root.MergeBranch(r.Context(), poolID, childBranch, parentBranch, message.Author, message.Body)
	if err != nil {
		w.Error(err)
		return
	}
	c.publishEvent("branch-merge", api.EventBranchCommit{
		CommitID: commit.String(),
		PoolID:   poolID.String(),
		Branch:   childBranch,
		Parent:   parentBranch,
	})
	w.Respond(http.StatusOK, api.CommitResponse{Commit: commit})
}

func handlePoolDelete(c *Core, w *ResponseWriter, r *Request) {
	id, ok := r.PoolID(w, c.root)
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
	poolID, ok := r.PoolID(w, c.root)
	if !ok {
		return
	}
	branchName, ok := r.StringFromPath(w, "branch")
	if !ok {
		return
	}
	if err := c.root.RemoveBranch(r.Context(), poolID, branchName); err != nil {
		w.Error(err)
		return
	}
	c.publishEvent("branch-delete", api.EventBranch{
		PoolID: poolID.String(),
		Branch: branchName,
	})
	w.WriteHeader(http.StatusNoContent)
}

type warningCollector []string

func (w *warningCollector) Warn(msg string) error {
	*w = append(*w, msg)
	return nil
}

func handleBranchLoad(c *Core, w *ResponseWriter, r *Request) {
	poolID, ok := r.PoolID(w, c.root)
	if !ok {
		return
	}
	branchName, ok := r.StringFromPath(w, "branch")
	if !ok {
		return
	}
	message, ok := r.decodeCommitMessage(w)
	if !ok {
		return
	}
	pool, err := c.root.OpenPool(r.Context(), poolID)
	if err != nil {
		w.Error(err)
		return
	}
	branch, err := pool.OpenBranchByName(r.Context(), branchName)
	if err != nil {
		w.Error(err)
		return
	}
	// Force validation of ZNG when initialing loading into the lake.
	var opts anyio.ReaderOpts
	opts.Zng.Validate = true
	zr, err := anyio.NewReaderWithOpts(anyio.GzipReader(r.Body), zed.NewContext(), opts)
	if err != nil {
		w.Error(err)
		return
	}
	warnings := warningCollector{}
	zr = zio.NewWarningReader(zr, &warnings)
	kommit, err := branch.Load(r.Context(), zr, message.Author, message.Body)
	if err != nil {
		if errors.Is(err, commits.ErrEmptyTransaction) {
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
		Branch:   branch.Name,
	})
}

func handleDelete(c *Core, w *ResponseWriter, r *Request) {
	poolID, ok := r.PoolID(w, c.root)
	if !ok {
		return
	}
	branchName, ok := r.StringFromPath(w, "branch")
	if !ok {
		return
	}
	message, ok := r.decodeCommitMessage(w)
	if !ok {
		return
	}
	var payload api.DeleteRequest
	if !r.Unmarshal(w, &payload) {
		return
	}
	pool, err := c.root.OpenPool(r.Context(), poolID)
	if err != nil {
		w.Error(err)
		return
	}
	branch, err := pool.OpenBranchByName(r.Context(), branchName)
	if err != nil {
		w.Error(err)
		return
	}
	ids, err := branch.LookupTags(r.Context(), payload.ObjectIDs)
	if err != nil {
		w.Error(err)
		return
	}
	commit, err := branch.Delete(r.Context(), ids, message.Author, message.Body)
	if err != nil {
		w.Error(err)
		return
	}
	w.Marshal(api.CommitResponse{Commit: commit})
}

func handleIndexRulesPost(c *Core, w *ResponseWriter, r *Request) {
	var body api.IndexRulesAddRequest
	if !r.Unmarshal(w, &body, index.RuleTypes...) {
		return
	}
	if err := c.root.AddIndexRules(r.Context(), body.Rules); err != nil {
		w.Error(err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleIndexRulesDelete(c *Core, w *ResponseWriter, r *Request) {
	var req api.IndexRulesDeleteRequest
	if !r.Unmarshal(w, &req) {
		return
	}
	ruleIDs, err := lakeparse.ParseIDs(req.RuleIDs)
	if err != nil {
		w.Error(zqe.ErrInvalid(err))
	}
	rules, err := c.root.DeleteIndexRules(r.Context(), ruleIDs)
	if err != nil {
		w.Error(err)
		return
	}
	w.Respond(http.StatusOK, api.IndexRulesDeleteResponse{Rules: rules})
}

func handleIndexApply(c *Core, w *ResponseWriter, r *Request, branch *lake.Branch) {
	var req api.IndexApplyRequest
	if !r.Unmarshal(w, &req) {
		return
	}
	tags, err := branch.LookupTags(r.Context(), req.Tags)
	if err != nil {
		w.Error(err)
		return
	}
	rules, err := c.root.LookupIndexRules(r.Context(), req.RuleName)
	if err != nil {
		w.Error(err)
		return
	}
	commit, err := branch.ApplyIndexRules(r.Context(), rules, tags)
	if err != nil {
		w.Error(err)
		return
	}
	w.Respond(http.StatusOK, api.CommitResponse{Commit: commit})

}

func handleIndexUpdate(c *Core, w *ResponseWriter, r *Request, branch *lake.Branch) {
	var req api.IndexUpdateRequest
	if !r.Unmarshal(w, &req) {
		return
	}
	var err error
	var rules []index.Rule
	if len(req.RuleNames) > 0 {
		rules, err = c.root.LookupIndexRules(r.Context(), req.RuleNames...)
	} else {
		rules, err = c.root.AllIndexRules(r.Context())
	}
	if err != nil {
		w.Error(err)
		return
	}
	commit, err := branch.UpdateIndex(r.Context(), rules)
	if err != nil {
		if errors.Is(err, commits.ErrEmptyTransaction) {
			err = zqe.ErrInvalid(err)
		}
		w.Error(err)
		return
	}
	w.Respond(http.StatusOK, api.CommitResponse{Commit: commit})
}

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

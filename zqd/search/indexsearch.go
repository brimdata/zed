package search

import (
	"context"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zqd/api"
)

type IndexSearcher interface {
	IndexSearch(context.Context, archive.IndexQuery) (zbuf.ReadCloser, error)
}

type IndexSearchOp struct {
	zbuf.ReadCloser
}

func NewIndexSearchOp(ctx context.Context, s IndexSearcher, req api.IndexSearchRequest) (*IndexSearchOp, error) {
	query, err := archive.ParseIndexQuery(req.IndexName, req.Patterns)
	if err != nil {
		return nil, err
	}
	rc, err := s.IndexSearch(ctx, query)
	if err != nil {
		return nil, err
	}
	return &IndexSearchOp{rc}, nil
}

func (s *IndexSearchOp) Run(out Output) (err error) {
	if err = out.SendControl(&api.TaskStart{"TaskStart", 0}); err != nil {
		return
	}
	defer func() {
		if err != nil {
			verr := api.Error{Type: "INTERNAL", Message: err.Error()}
			out.End(&api.TaskEnd{"TaskEnd", 0, &verr})
			return
		}
		err = out.End(&api.TaskEnd{"TaskEnd", 0, nil})
	}()

	for {
		var b zbuf.Batch
		if b, err = zbuf.ReadBatch(s, DefaultMTU); err != nil {
			return
		}
		if b == nil {
			return
		}
		if err = out.SendBatch(0, b); err != nil {
			return
		}
	}
}

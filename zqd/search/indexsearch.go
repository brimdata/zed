package search

import (
	"context"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zqd/api"
)

type IndexSearch struct {
	zbuf.ReadCloser
}

type IndexedArchive interface {
	IndexSearch(context.Context, archive.IndexQuery) (zbuf.ReadCloser, error)
}

func NewIndexSearch(ctx context.Context, s IndexedArchive, req api.IndexSearchRequest) (*IndexSearch, error) {
	query, err := archive.ParseIndexQuery(req.IndexName, req.Patterns)
	if err != nil {
		return nil, err
	}
	rc, err := s.IndexSearch(ctx, query)
	if err != nil {
		return nil, err
	}
	return &IndexSearch{rc}, nil
}

func (s *IndexSearch) Run(out Output) (err error) {
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

package search

import (
	"context"

	"github.com/brimsec/zq/archive"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zqd/api"
	"github.com/brimsec/zq/zqd/space"
	"github.com/brimsec/zq/zqd/storage/archivestore"
	"github.com/brimsec/zq/zqe"
)

type IndexSearch struct {
	zbuf.ReadCloser
}

func NewIndexSearch(ctx context.Context, s *space.Space, req api.IndexSearchRequest) (*IndexSearch, error) {
	arkstore, ok := s.Storage.(*archivestore.Storage)
	if !ok {
		return nil, zqe.E(zqe.Invalid, "index search only supported on archive spaces")
	}

	query, err := archive.ParseIndexQuery(req.IndexName, req.Patterns)
	if err != nil {
		return nil, err
	}
	rc, err := arkstore.IndexSearch(ctx, query)
	if err != nil {
		return nil, err
	}
	return &IndexSearch{rc}, nil
}

func (s *IndexSearch) Run(out Output) (err error) {
	id := int64(0)
	if err = out.SendControl(&api.TaskStart{"TaskStart", id}); err != nil {
		return
	}
	defer func() {
		if err != nil {
			verr := api.Error{Type: "INTERNAL", Message: err.Error()}
			out.End(&api.TaskEnd{"TaskEnd", id, &verr})
			return
		}
		err = out.End(&api.TaskEnd{"TaskEnd", id, nil})
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

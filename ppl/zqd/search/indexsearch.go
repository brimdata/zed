package search

import (
	"context"

	"github.com/brimdata/zq/api"
	"github.com/brimdata/zq/ppl/lake/index"
	"github.com/brimdata/zq/zbuf"
	"github.com/brimdata/zq/zng/resolver"
)

type IndexSearcher interface {
	IndexSearch(context.Context, *resolver.Context, index.Query) (zbuf.ReadCloser, error)
}

type IndexSearchOp struct {
	zbuf.ReadCloser
}

func NewIndexSearchOp(ctx context.Context, s IndexSearcher, req api.IndexSearchRequest) (*IndexSearchOp, error) {
	query, err := index.ParseQuery(req.IndexName, req.Patterns)
	if err != nil {
		return nil, err
	}
	rc, err := s.IndexSearch(ctx, resolver.NewContext(), query)
	if err != nil {
		return nil, err
	}
	return &IndexSearchOp{rc}, nil
}

func (s *IndexSearchOp) Run(out Output) (err error) {
	return SendFromReader(out, s)
}

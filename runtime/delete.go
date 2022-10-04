package runtime

import (
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zbuf"
	"github.com/segmentio/ksuid"
)

type DeleteQuery struct {
	*Query
	DeleteMeter
}

type DeleteMeter interface {
	DeleteObjects() []ksuid.KSUID
}

func NewDeleteQuery(pctx *op.Context, puller zbuf.Puller, meter DeleteMeter) *DeleteQuery {
	return &DeleteQuery{
		Query:       NewQuery(pctx, puller, nil),
		DeleteMeter: meter,
	}
}

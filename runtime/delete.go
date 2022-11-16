package runtime

import (
	"sync"

	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zbuf"
	"github.com/segmentio/ksuid"
)

type DeleteQuery struct {
	*Query
	deletes *sync.Map
}

func NewDeleteQuery(pctx *op.Context, puller zbuf.Puller, deletes *sync.Map) *DeleteQuery {
	return &DeleteQuery{
		Query:   NewQuery(pctx, puller, nil),
		deletes: deletes,
	}
}

func (d *DeleteQuery) DeletionSet() []ksuid.KSUID {
	var ids []ksuid.KSUID
	if d.deletes != nil {
		d.deletes.Range(func(key, value any) bool {
			ids = append(ids, key.(ksuid.KSUID))
			return true
		})
	}
	return ids
}

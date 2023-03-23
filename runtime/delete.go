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

func NewDeleteQuery(octx *op.Context, puller zbuf.Puller, deletes *sync.Map) *DeleteQuery {
	return &DeleteQuery{
		Query:   NewQuery(octx, puller, nil),
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

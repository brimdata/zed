package reducer

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
)

type Count struct {
	Reducer
	Field string
	count uint64
}

func NewCount(name, field string) *Count {
	return &Count{
		Reducer: New(name),
		Field:   field,
	}
}

func (c *Count) Consume(r *zson.Record) {
	if c.Field != "" {
		if _, ok := r.ColumnOfField(c.Field); !ok {
			return
		}
	}
	c.count++
}

func (c *Count) Result() zeek.Value {
	return &zeek.Count{c.count}
}

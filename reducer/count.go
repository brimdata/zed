package reducer

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zq"
)

type CountProto struct {
	target string
	field  string
}

func (cp *CountProto) Target() string {
	return cp.target
}

func (cp *CountProto) Instantiate() Interface {
	return &Count{Field: cp.field}
}

func NewCountProto(target, field string) *CountProto {
	return &CountProto{target, field}
}

type Count struct {
	Reducer
	Field string
	count uint64
}

func (c *Count) Consume(r *zq.Record) {
	if c.Field != "" {
		if _, ok := r.ColumnOfField(c.Field); !ok {
			return
		}
	}
	c.count++
}

func (c *Count) Result() zeek.Value {
	return zeek.NewCount(c.count)
}

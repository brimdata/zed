package reducer

import (
	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zng"
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

func (c *Count) Consume(r *zbuf.Record) {
	if c.Field != "" {
		if _, ok := r.ColumnOfField(c.Field); !ok {
			return
		}
	}
	c.count++
}

func (c *Count) Result() zng.Value {
	return zng.NewCount(c.count)
}

package field

import (
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"github.com/mccanne/zq/streamfn"
)

type Int struct {
	fn *streamfn.Int64
}

func NewIntStreamfn(op string) Streamfn {
	return &Int{
		fn: streamfn.NewInt64(op),
	}
}

func (i *Int) Result() zeek.Value {
	return &zeek.Int{i.fn.State}
}

func (i *Int) Consume(v zeek.Value) error {
	cv := zeek.CoerceToInt(v)
	if cv == nil {
		return zson.ErrTypeMismatch
	}
	i.fn.Update(cv.Native)
	return nil
}

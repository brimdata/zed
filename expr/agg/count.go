package agg

import (
	"github.com/brimdata/zq/zng"
	"github.com/brimdata/zq/zng/resolver"
)

type Count uint64

func (c *Count) Consume(v zng.Value) error {
	if !v.IsNil() {
		*c++
	}
	return nil
}

func (c Count) Result(*resolver.Context) (zng.Value, error) {
	return zng.NewUint64(uint64(c)), nil
}

func (c *Count) ConsumeAsPartial(p zng.Value) error {
	u, err := zng.DecodeUint(p.Bytes)
	if err == nil {
		*c += Count(u)
	}
	return err
}

func (c Count) ResultAsPartial(*resolver.Context) (zng.Value, error) {
	return c.Result(nil)
}

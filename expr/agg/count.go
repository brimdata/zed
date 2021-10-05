package agg

import (
	"github.com/brimdata/zed"
)

type Count uint64

func (c *Count) Consume(v zed.Value) error {
	if !v.IsNil() {
		*c++
	}
	return nil
}

func (c Count) Result(*zed.Context) (zed.Value, error) {
	return zed.NewUint64(uint64(c)), nil
}

func (c *Count) ConsumeAsPartial(p zed.Value) error {
	u, err := zed.DecodeUint(p.Bytes)
	if err == nil {
		*c += Count(u)
	}
	return err
}

func (c Count) ResultAsPartial(*zed.Context) (zed.Value, error) {
	return c.Result(nil)
}

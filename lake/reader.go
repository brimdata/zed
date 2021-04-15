package lake

import (
	"github.com/brimdata/zed/zson"
)

func NewPoolConfigReader(pools []PoolConfig) *zson.MarshalStream {
	reader := zson.NewMarshalStream(zson.StyleSimple)
	go func() {
		for k := range pools {
			if !reader.Supply(&pools[k]) {
				return
			}
		}
		reader.Close(nil)
	}()
	return reader
}

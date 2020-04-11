package zdx_test

import (
	"strings"
	"testing"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zdx"
	"github.com/brimsec/zq/zio/zngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const stream1 = `
#0:record[key:string,value:string]
0:[1;a;]
0:[3;b;]
0:[5;c;]`

const stream2 = `
#0:record[key:string,value:string]
0:[2;d;]
0:[4;e;]
0:[6;f;]`

func TestCombinerOrder(t *testing.T) {
	zctx := resolver.NewContext()
	s1 := zngio.NewReader(strings.NewReader(stream1), zctx)
	s2 := zngio.NewReader(strings.NewReader(stream2), zctx)
	c := zdx.NewCombiner([]zbuf.Reader{s1, s2}, func(a, b *zng.Record) *zng.Record {
		return a
	})
	var keys []string
	for {
		rec, _ := c.Read()
		if rec == nil {
			break
		}
		key, err := rec.AccessString("key")
		require.NoError(t, err)
		keys = append(keys, key)
	}
	assert.Equal(t, []string{"1", "2", "3", "4", "5", "6"}, keys)
}

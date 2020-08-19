package proc_test

import (
	"bytes"
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/brimsec/zq/pkg/test"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordPuller is a proc.Proc whose Pull method returns one batch for each
// record of a zbuf.Reader.
type recordPuller struct {
	proc.Base
	r zbuf.Reader
}

func (rp *recordPuller) Pull() (zbuf.Batch, error) {
	for {
		rec, err := rp.r.Read()
		if rec == nil || err != nil {
			return nil, err
		}
		return zbuf.NewArray([]*zng.Record{rec}), nil
	}
}

var omTestInputs = []string{
	`
#0:record[v:int32,ts:time]
0:[10;1;]
0:[20;2;]
0:[30;3;]`,
	`
#0:record[v:int32,ts:time]
0:[15;4;]
0:[25;5;]
0:[35;6;]`,
}

func TestParallelOrder(t *testing.T) {
	cases := []struct {
		field    string
		reversed bool
		exp      string
	}{
		{
			field:    "ts",
			reversed: false,
			exp: `
#0:record[v:int32,ts:time]
0:[10;1;]
0:[20;2;]
0:[30;3;]
0:[15;4;]
0:[25;5;]
0:[35;6;]
`,
		},
		{
			field:    "v",
			reversed: false,
			exp: `
#0:record[v:int32,ts:time]
0:[10;1;]
0:[15;4;]
0:[20;2;]
0:[25;5;]
0:[30;3;]
0:[35;6;]
`,
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			zctx := resolver.NewContext()
			pctx := &proc.Context{Context: context.Background(), TypeContext: zctx}
			var parents []proc.Proc
			for _, s := range omTestInputs {
				r := tzngio.NewReader(bytes.NewReader([]byte(s)), zctx)
				parents = append(parents, &recordPuller{Base: proc.Base{Context: pctx}, r: r})
			}
			om := proc.NewOrderedMerge(pctx, parents, c.field, c.reversed)

			var sb strings.Builder
			err := zbuf.CopyPuller(tzngio.NewWriter(&sb), om)
			require.NoError(t, err)
			assert.Equal(t, test.Trim(c.exp), sb.String())
		})
	}
}

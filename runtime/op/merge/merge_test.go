package merge_test

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/runtime/op/merge"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var omTestInputs = []string{
	`
{v:10,ts:1970-01-01T00:00:01Z}
{v:20,ts:1970-01-01T00:00:02Z}
{v:30,ts:1970-01-01T00:00:03Z}
`,
	`
{v:15,ts:1970-01-01T00:00:04Z}
{v:25,ts:1970-01-01T00:00:05Z}
{v:35,ts:1970-01-01T00:00:06Z}
`,
}

var omTestInputRev = []string{
	`
{v:30,ts:1970-01-01T00:00:03Z}
{v:20,ts:1970-01-01T00:00:02Z}
{v:10,ts:1970-01-01T00:00:01Z}
`,
	`
{v:35,ts:1970-01-01T00:00:06Z}
{v:25,ts:1970-01-01T00:00:05Z}
{v:15,ts:1970-01-01T00:00:04Z}
`,
}

func TestParallelOrder(t *testing.T) {
	cases := []struct {
		field  string
		order  order.Which
		inputs []string
		exp    string
	}{
		{
			field:  "ts",
			order:  order.Asc,
			inputs: omTestInputs,
			exp: `
{v:10,ts:1970-01-01T00:00:01Z}
{v:20,ts:1970-01-01T00:00:02Z}
{v:30,ts:1970-01-01T00:00:03Z}
{v:15,ts:1970-01-01T00:00:04Z}
{v:25,ts:1970-01-01T00:00:05Z}
{v:35,ts:1970-01-01T00:00:06Z}
`,
		},
		{

			field:  "v",
			order:  order.Asc,
			inputs: omTestInputs,
			exp: `
{v:10,ts:1970-01-01T00:00:01Z}
{v:15,ts:1970-01-01T00:00:04Z}
{v:20,ts:1970-01-01T00:00:02Z}
{v:25,ts:1970-01-01T00:00:05Z}
{v:30,ts:1970-01-01T00:00:03Z}
{v:35,ts:1970-01-01T00:00:06Z}
`,
		},
		{
			field:  "ts",
			order:  order.Desc,
			inputs: omTestInputRev,
			exp: `
{v:35,ts:1970-01-01T00:00:06Z}
{v:25,ts:1970-01-01T00:00:05Z}
{v:15,ts:1970-01-01T00:00:04Z}
{v:30,ts:1970-01-01T00:00:03Z}
{v:20,ts:1970-01-01T00:00:02Z}
{v:10,ts:1970-01-01T00:00:01Z}
`,
		},
	}

	for i, c := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			zctx := zed.NewContext()
			pctx := &op.Context{Context: context.Background(), Zctx: zctx}
			var parents []zbuf.Puller
			for _, input := range c.inputs {
				r := zsonio.NewReader(strings.NewReader(input), zctx)
				parents = append(parents, zbuf.NewPuller(r, 10))
			}
			layout := order.NewLayout(c.order, field.DottedList(c.field))
			cmp := zbuf.NewComparator(zctx, layout).Compare
			om := merge.New(pctx.Context, parents, cmp)

			var sb strings.Builder
			err := zbuf.CopyPuller(zsonio.NewWriter(zio.NopCloser(&sb), zsonio.WriterOpts{}), om)
			require.NoError(t, err)
			assert.Equal(t, c.exp, "\n"+sb.String())
		})
	}
}

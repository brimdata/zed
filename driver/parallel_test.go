package driver

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/brimsec/zq/scanner"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var parallelTestInputs []string = []string{
	`
#0:record[v:int32,ts:time]
0:[0;0;]`,
	`
#0:record[v:int32,ts:time]
0:[1;1;]`,
	`
#0:record[v:int32,ts:time]
0:[2;2;]`,
	`
#0:record[v:int32,ts:time]
0:[3;3;]`,
	`
#0:record[v:int32,ts:time]
0:[4;4;]`,
}

type orderedmsrc struct{}

func (m *orderedmsrc) OrderInfo() (string, bool) {
	return "ts", false
}

func (m *orderedmsrc) SendSources(ctx context.Context, zctx *resolver.Context, sf SourceFilter, srcChan chan SourceOpener) error {
	// Create SourceOpeners that await a signal before returning, then
	// signal them in reverse of expected order.
	var releaseChs []chan struct{}
	for _ = range parallelTestInputs {
		releaseChs = append(releaseChs, make(chan struct{}))
	}
	for i := range parallelTestInputs {
		i := i
		rdr := tzngio.NewReader(strings.NewReader(parallelTestInputs[i]), zctx)
		sn, err := scanner.NewScanner(ctx, rdr, sf.Filter, sf.FilterExpr, sf.Span)
		if err != nil {
			return err
		}
		srcChan <- func() (ScannerCloser, error) {
			select {
			case <-releaseChs[i]:
			}
			return &noCloseScanner{
				Scanner: sn,
			}, nil
		}
	}
	for i := len(parallelTestInputs) - 1; i >= 0; i-- {
		close(releaseChs[i])
	}
	return nil
}

func trim(s string) string {
	return strings.TrimSpace(s) + "\n"
}

func TestParallelOrder(t *testing.T) {
	t.Parallel()

	// Use `v!=3` to trigger & verify empty rank handling in orderedWaiter.
	query, err := zql.ParseProc("v!=3")
	require.NoError(t, err)

	var buf bytes.Buffer
	d := NewCLI(tzngio.NewWriter(&buf))
	zctx := resolver.NewContext()
	err = MultiRun(context.Background(), d, query, zctx, &orderedmsrc{}, MultiConfig{
		Parallelism: len(parallelTestInputs),
	})
	require.NoError(t, err)

	exp := `
#0:record[v:int32,ts:time]
0:[0;0;]
0:[1;1;]
0:[2;2;]
0:[4;4;]
`
	assert.Equal(t, trim(exp), buf.String())
}

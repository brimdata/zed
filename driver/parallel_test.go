package driver

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/brimsec/zq/multisource"
	"github.com/brimsec/zq/scanner"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/brimsec/zq/zql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type scannerCloser struct {
	scanner.Scanner
	io.Closer
}

type onClose struct {
	fn func() error
}

func (c *onClose) Close() error {
	if c.fn == nil {
		return nil
	}
	return c.fn()
}

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

func (m *orderedmsrc) SendSources(ctx context.Context, zctx *resolver.Context, sf multisource.SourceFilter, srcChan chan multisource.Source) error {
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
		srcChan <- func() (multisource.ScannerCloser, error) {
			select {
			case <-releaseChs[i]:
			}
			return &scannerCloser{
				Scanner: sn,
				Closer:  &onClose{},
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
	d := NewCLI(tzngio.NewWriter(zio.NopCloser(&buf)))
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

// A noEndScanner never returns proc.EOS from its Pull().
type noEndScanner struct {
	input string
	zctx  *resolver.Context
}

func (rp *noEndScanner) Pull() (zbuf.Batch, error) {
	r := tzngio.NewReader(strings.NewReader(rp.input), rp.zctx)
	return zbuf.ReadBatch(r, 1)
}

func (rp *noEndScanner) Stats() *scanner.ScannerStats {
	return &scanner.ScannerStats{}
}

type scannerCloseMS struct {
	closed chan struct{}
	input  string
}

func (m *scannerCloseMS) OrderInfo() (string, bool) {
	return "", false
}

func (m *scannerCloseMS) SendSources(ctx context.Context, zctx *resolver.Context, sf multisource.SourceFilter, srcChan chan multisource.Source) error {
	srcChan <- func() (multisource.ScannerCloser, error) {
		return &scannerCloser{
			// Use a noEndScanner so that a parallel head never tries to
			// close the ScannerCloser in its Pull. That way, if the Close fires,
			// we know it must have happened due to the query context cancellation.
			Scanner: &noEndScanner{input: m.input, zctx: zctx},
			Closer: &onClose{fn: func() error {
				close(m.closed)
				return nil
			}},
		}, nil
	}
	return nil
}

// TestScannerClose verifies that any open ScannerCloser's will be closed soon
// after the MultiRun call finishes.
func TestScannerClose(t *testing.T) {
	query, err := zql.ParseProc("* | head 1")
	require.NoError(t, err)

	var buf bytes.Buffer
	d := NewCLI(tzngio.NewWriter(zio.NopCloser(&buf)))
	zctx := resolver.NewContext()
	ms := &scannerCloseMS{
		input: `
#0:record[v:int32,ts:time]
0:[1;1;]
`,
		closed: make(chan struct{}),
	}
	err = MultiRun(context.Background(), d, query, zctx, ms, MultiConfig{})
	require.NoError(t, err)
	require.Equal(t, trim(ms.input), trim(buf.String()))

	tm := time.NewTimer(5 * time.Second)
	select {
	case <-ms.closed:
	case <-tm.C:
		t.Fatal("time out waiting for close")
	}
}

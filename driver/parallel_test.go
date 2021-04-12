package driver

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/brimdata/zed/zson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type scannerCloser struct {
	zbuf.Scanner
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

type testSource struct {
	opener func(*zson.Context, SourceFilter) (ScannerCloser, error)
}

func (s *testSource) Open(ctx context.Context, zctx *zson.Context, sf SourceFilter) (ScannerCloser, error) {
	return s.opener(zctx, sf)
}

func (s *testSource) ToRequest(*api.WorkerChunkRequest) error {
	return errors.New("ToRequest called on testSource")
}

var parallelTestInputs = []string{
	"{v:0 (int32),ts:1970-01-01T00:00:00Z} (=0)",
	"{v:1 (int32),ts:1970-01-01T00:00:01Z} (=0)",
	"{v:2 (int32),ts:1970-01-01T00:00:02Z} (=0)",
	"{v:3 (int32),ts:1970-01-01T00:00:03Z} (=0)",
	"{v:4 (int32),ts:1970-01-01T00:00:04Z} (=0)",
}

type orderedmsrc struct{}

func (m *orderedmsrc) OrderInfo() (field.Static, bool) {
	return field.New("ts"), false
}

func (m *orderedmsrc) SendSources(ctx context.Context, span nano.Span, srcChan chan Source) error {
	// Create SourceOpeners that await a signal before returning, then
	// signal them in reverse of expected order.
	var releaseChs []chan struct{}
	for _ = range parallelTestInputs {
		releaseChs = append(releaseChs, make(chan struct{}))
	}
	for i := range parallelTestInputs {
		i := i
		opener := func(zctx *zson.Context, sf SourceFilter) (ScannerCloser, error) {
			rdr := zson.NewReader(strings.NewReader(parallelTestInputs[i]), zctx)
			sn, err := zbuf.NewScanner(ctx, rdr, sf.Filter, sf.Span)
			if err != nil {
				return nil, err
			}
			select {
			case <-releaseChs[i]:
			}
			return &scannerCloser{
				Scanner: sn,
				Closer:  &onClose{},
			}, nil
		}
		srcChan <- &testSource{opener: opener}
	}
	for i := len(parallelTestInputs) - 1; i >= 0; i-- {
		close(releaseChs[i])
	}
	return nil
}

func (m *orderedmsrc) SourceFromRequest(context.Context, *api.WorkerChunkRequest) (Source, error) {
	return nil, errors.New("SourceFromRequest called on orderedmsrc")
}

func trim(s string) string {
	return strings.TrimSpace(s) + "\n"
}

func TestParallelOrder(t *testing.T) {
	t.Parallel()

	// Use `v!=3` to trigger & verify empty rank handling in orderedWaiter.
	query, err := compiler.ParseProc("v!=3")
	require.NoError(t, err)

	var buf bytes.Buffer
	d := NewCLI(zsonio.NewWriter(zio.NopCloser(&buf), zsonio.WriterOpts{}))
	zctx := zson.NewContext()
	err = MultiRun(context.Background(), d, query, zctx, &orderedmsrc{}, MultiConfig{
		Parallelism: len(parallelTestInputs),
	})
	require.NoError(t, err)

	const exp = `
{v:0 (int32),ts:1970-01-01T00:00:00Z} (=0)
{v:1,ts:1970-01-01T00:00:01Z} (0)
{v:2,ts:1970-01-01T00:00:02Z} (0)
{v:4,ts:1970-01-01T00:00:04Z} (0)
`
	assert.Equal(t, trim(exp), buf.String())
}

// A noEndScanner never returns proc.EOS from its Pull().
type noEndScanner struct {
	input string
	zctx  *zson.Context
}

func (rp *noEndScanner) Pull() (zbuf.Batch, error) {
	r := zson.NewReader(strings.NewReader(rp.input), rp.zctx)
	return zbuf.NewPuller(r, 1).Pull()
}

func (rp *noEndScanner) Stats() *zbuf.ScannerStats {
	return &zbuf.ScannerStats{}
}

type scannerCloseMS struct {
	closed chan struct{}
	input  string
}

func (m *scannerCloseMS) OrderInfo() (field.Static, bool) {
	return nil, false
}

func (m *scannerCloseMS) SendSources(ctx context.Context, span nano.Span, srcChan chan Source) error {
	opener := func(zctx *zson.Context, _ SourceFilter) (ScannerCloser, error) {
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
	srcChan <- &testSource{opener: opener}
	return nil
}

func (m *scannerCloseMS) SourceFromRequest(context.Context, *api.WorkerChunkRequest) (Source, error) {
	return nil, errors.New("SourceFromRequest called on scannerCloseMS")
}

// TestScannerClose verifies that any open ScannerCloser's will be closed soon
// after the MultiRun call finishes.
func TestScannerClose(t *testing.T) {
	query, err := compiler.ParseProc("* | head 1")
	require.NoError(t, err)

	var buf bytes.Buffer
	d := NewCLI(zsonio.NewWriter(zio.NopCloser(&buf), zsonio.WriterOpts{}))
	zctx := zson.NewContext()
	ms := &scannerCloseMS{
		input:  "{v:1 (int32),ts:1970-01-01T00:00:01Z} (=0)",
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

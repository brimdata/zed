package runtime

import (
	"context"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
)

// Query runs a flowgraph as a zbuf.Puller and implements a Close() method
// that gracefully tears down the flowgraph.  Its AsReader() and AsProgressReader()
// methods provide a convenient means to run a flowgraph as zio.Reader.
type Query struct {
	zbuf.Puller
	octx  *op.Context
	meter zbuf.Meter
}

var _ zbuf.Puller = (*Query)(nil)

func NewQuery(octx *op.Context, puller zbuf.Puller, meter zbuf.Meter) *Query {
	return &Query{
		Puller: puller,
		octx:   octx,
		meter:  meter,
	}
}

type Compiler interface {
	NewQuery(*op.Context, ast.Seq, []zio.Reader) (*Query, error)
	NewLakeQuery(*op.Context, ast.Seq, int, *lakeparse.Commitish) (*Query, error)
	NewLakeDeleteQuery(*op.Context, ast.Seq, *lakeparse.Commitish) (*DeleteQuery, error)
	Parse(string, ...string) (ast.Seq, error)
}

func CompileQuery(ctx context.Context, zctx *zed.Context, c Compiler, program ast.Seq, readers []zio.Reader) (*Query, error) {
	octx := op.NewContext(ctx, zctx)
	q, err := c.NewQuery(octx, program, readers)
	if err != nil {
		octx.Cancel()
		return nil, err
	}
	return q, nil
}

func CompileLakeQuery(ctx context.Context, zctx *zed.Context, c Compiler, program ast.Seq, head *lakeparse.Commitish) (*Query, error) {
	octx := op.NewContext(ctx, zctx)
	q, err := c.NewLakeQuery(octx, program, 0, head)
	if err != nil {
		octx.Cancel()
		return nil, err
	}
	return q, nil
}

func (q *Query) AsReader() zio.Reader {
	return zbuf.PullerReader(q)
}

func (q *Query) AsProgressReadCloser() zbuf.ProgressReadCloser {
	return struct {
		zio.Reader
		io.Closer
		zbuf.Meter
	}{q.AsReader(), q, q}
}

func (q *Query) Progress() zbuf.Progress {
	return q.meter.Progress()
}

func (q *Query) Meter() zbuf.Meter {
	return q.meter
}

func (q *Query) Close() error {
	q.octx.Cancel()
	return nil
}

func (q *Query) Pull(done bool) (zbuf.Batch, error) {
	if done {
		q.octx.Cancel()
	}
	return q.Puller.Pull(done)
}

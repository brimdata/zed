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
	"go.uber.org/zap"
)

// Query runs a flowgraph as a zbuf.Puller and implements a Close() method
// that gracefully tears down the flowgraph.  Its AsReader() and AsProgressReader()
// methods provide a convenient means to run a flowgraph as zio.Reader.
type Query struct {
	zbuf.Puller
	pctx  *op.Context
	meter *meter
}

var _ zbuf.Puller = (*Query)(nil)

func NewQuery(pctx *op.Context, puller zbuf.Puller, meters []zbuf.Meter) *Query {
	return &Query{
		Puller: puller,
		pctx:   pctx,
		meter:  &meter{meters},
	}
}

type Compiler interface {
	NewQuery(*op.Context, ast.Op, []zio.Reader) (*Query, error)
	NewLakeQuery(*op.Context, ast.Op, int, *lakeparse.Commitish) (*Query, error)
	NewLakeDeleteQuery(*op.Context, ast.Op, *lakeparse.Commitish) (*DeleteQuery, error)
	Parse(string, ...string) (ast.Op, error)
}

func CompileQuery(ctx context.Context, zctx *zed.Context, c Compiler, program ast.Op, readers []zio.Reader) (*Query, error) {
	pctx := op.NewContext(ctx, zctx, nil)
	q, err := c.NewQuery(pctx, program, readers)
	if err != nil {
		pctx.Cancel()
		return nil, err
	}
	return q, nil
}

func CompileLakeQuery(ctx context.Context, zctx *zed.Context, c Compiler, program ast.Op, head *lakeparse.Commitish, logger *zap.Logger) (*Query, error) {
	pctx := op.NewContext(ctx, zctx, logger)
	q, err := c.NewLakeQuery(pctx, program, 0, head)
	if err != nil {
		pctx.Cancel()
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
	q.pctx.Cancel()
	return nil
}

func (q *Query) Pull(done bool) (zbuf.Batch, error) {
	if done {
		q.pctx.Cancel()
	}
	return q.Puller.Pull(done)
}

type meter struct {
	meters []zbuf.Meter
}

func (m *meter) Progress() zbuf.Progress {
	var out zbuf.Progress
	for _, meter := range m.meters {
		out.Add(meter.Progress())
	}
	return out
}

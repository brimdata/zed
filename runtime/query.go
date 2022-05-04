package runtime

import (
	"context"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/order"
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
	pctx      *op.Context
	flowgraph *compiler.Runtime
}

var _ zbuf.Puller = (*Query)(nil)

func NewQuery(pctx *op.Context, flowgraph *compiler.Runtime, closer io.Closer) *Query {
	return &Query{
		Puller:    flowgraph.Puller(),
		pctx:      pctx,
		flowgraph: flowgraph,
	}
}

func NewQueryOnReader(ctx context.Context, zctx *zed.Context, program ast.Op, reader zio.Reader, logger *zap.Logger) (*Query, error) {
	pctx := op.NewContext(ctx, zctx, logger)
	flowgraph, err := compiler.CompileForInternal(pctx, program, reader)
	if err != nil {
		pctx.Cancel()
		return nil, err
	}
	return NewQuery(pctx, flowgraph, nil), nil
}

func NewQueryOnOrderedReader(ctx context.Context, zctx *zed.Context, program ast.Op, reader zio.Reader, layout order.Layout, logger *zap.Logger) (*Query, error) {
	pctx := op.NewContext(ctx, zctx, logger)
	flowgraph, err := compiler.CompileForInternalWithOrder(pctx, program, reader, layout)
	if err != nil {
		pctx.Cancel()
		return nil, err
	}
	return NewQuery(pctx, flowgraph, nil), nil
}

func NewQueryOnFileSystem(ctx context.Context, zctx *zed.Context, program ast.Op, readers []zio.Reader, adaptor op.DataAdaptor) (*Query, error) {
	pctx := op.NewContext(ctx, zctx, nil)
	flowgraph, err := compiler.CompileForFileSystem(pctx, program, readers, adaptor)
	if err != nil {
		pctx.Cancel()
		return nil, err
	}
	return NewQuery(pctx, flowgraph, nil), nil
}

func NewQueryOnLake(ctx context.Context, zctx *zed.Context, program ast.Op, lake op.DataAdaptor, head *lakeparse.Commitish, logger *zap.Logger) (*Query, error) {
	pctx := op.NewContext(ctx, zctx, logger)
	flowgraph, err := compiler.CompileForLake(pctx, program, lake, 0, head)
	if err != nil {
		pctx.Cancel()
		return nil, err
	}
	return NewQuery(pctx, flowgraph, nil), nil
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
	return q.flowgraph.Meter().Progress()
}

func (q *Query) Meter() zbuf.Meter {
	return q.flowgraph.Meter()
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

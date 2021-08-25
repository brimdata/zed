package compiler

import (
	"context"
	"errors"
	"fmt"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

func CompileForInternal(pctx *proc.Context, p ast.Proc, r zio.Reader) (*Runtime, error) {
	return CompileForInternalWithOrder(pctx, p, r, order.Layout{})
}

func CompileForInternalWithOrder(pctx *proc.Context, p ast.Proc, r zio.Reader, layout order.Layout) (*Runtime, error) {
	adaptor := &internalAdaptor{}
	runtime, err := New(pctx, p, adaptor)
	if err != nil {
		return nil, err
	}
	readers := runtime.readers
	if len(readers) != 1 {
		return nil, fmt.Errorf("CompileForInternalWithOrder: Zed program expected %d readers", len(readers))
	}
	readers[0].Reader = r
	readers[0].Layout = layout
	return optimizeAndBuild(runtime)
}

type internalAdaptor struct{}

func (*internalAdaptor) LookupIDs(context.Context, string, string) (ksuid.KSUID, ksuid.KSUID, error) {
	return ksuid.Nil, ksuid.Nil, nil
}

func (*internalAdaptor) Layout(context.Context, dag.Source) order.Layout {
	return order.Nil
}

func (*internalAdaptor) NewScheduler(context.Context, *zson.Context, dag.Source, extent.Span, zbuf.Filter) (proc.Scheduler, error) {
	return nil, errors.New("invalid pool or file scan specified for internally streamed Zed query")
}

func (*internalAdaptor) Open(context.Context, *zson.Context, string, zbuf.Filter) (zbuf.PullerCloser, error) {
	return nil, errors.New("invalid file or URL access for internally streamed Zed query")
}

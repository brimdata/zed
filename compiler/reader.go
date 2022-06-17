package compiler

import (
	"context"
	"errors"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/runtime/expr/extent"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/segmentio/ksuid"
)

func NewCompiler() runtime.Compiler {
	return &anyCompiler{}
}

func (i *anyCompiler) NewQuery(pctx *op.Context, o ast.Op, readers []zio.Reader) (*runtime.Query, error) {
	if len(readers) != 1 {
		return nil, fmt.Errorf("NewQuery: Zed program expected %d readers", len(readers))
	}
	return i.CompileWithOrderDeprecated(pctx, o, readers[0], order.Layout{})
}

//XXX currently used only by group-by test, need to deprecate
func (*anyCompiler) CompileWithOrderDeprecated(pctx *op.Context, o ast.Op, r zio.Reader, layout order.Layout) (*runtime.Query, error) {
	adaptor := &internalAdaptor{}
	job, err := NewJob(pctx, o, adaptor, nil)
	if err != nil {
		return nil, err
	}
	readers := job.readers
	if len(readers) != 1 {
		return nil, fmt.Errorf("CompileForInternalWithOrder: Zed program expected %d readers", len(readers))
	}
	readers[0].Readers = []zio.Reader{r}
	readers[0].Layout = layout
	return optimizeAndBuild(job)
}

func (*anyCompiler) NewLakeQuery(pctx *op.Context, program ast.Op, parallelism int, head *lakeparse.Commitish) (*runtime.Query, error) {
	panic("NewLakeQuery called on compiler.anyCompiler")
}

type internalAdaptor struct{}

func (*internalAdaptor) CommitObject(context.Context, ksuid.KSUID, string) (ksuid.KSUID, error) {
	return ksuid.Nil, nil
}

func (*internalAdaptor) PoolID(context.Context, string) (ksuid.KSUID, error) {
	return ksuid.Nil, nil
}

func (*internalAdaptor) Layout(context.Context, dag.Source) order.Layout {
	return order.Nil
}

func (*internalAdaptor) NewScheduler(context.Context, *zed.Context, dag.Source, extent.Span, zbuf.Filter) (op.Scheduler, error) {
	return nil, errors.New("invalid pool or file scan specified for internally streamed Zed query")
}

func (*internalAdaptor) Open(context.Context, *zed.Context, string, string, zbuf.Filter) (zbuf.Puller, error) {
	return nil, errors.New("invalid file or URL access for internally streamed Zed query")
}

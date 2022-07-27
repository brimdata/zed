package compiler

import (
	"fmt"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/runtime/exec/querygen"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zio"
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
	job, err := NewJob(pctx, o, querygen.NewSource(nil, nil), nil)
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

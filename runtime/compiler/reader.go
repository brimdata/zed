package compiler

import (
	"fmt"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/semantic"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/runtime/exec/optimizer"
	"github.com/brimdata/zed/runtime/exec/querygen"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zio"
)

func NewCompiler() runtime.Compiler {
	return &compiler{}
}

func (i *compiler) NewQuery(pctx *op.Context, o ast.Op, readers []zio.Reader) (*runtime.Query, error) {
	if len(readers) != 1 {
		return nil, fmt.Errorf("NewQuery: Zed program expected %d readers", len(readers))
	}
	return i.CompileWithOrderDeprecated(pctx, o, readers[0], order.Layout{})
}

//XXX currently used only by group-by test, need to deprecate
func (*compiler) CompileWithOrderDeprecated(pctx *op.Context, program ast.Op, r zio.Reader, layout order.Layout) (*runtime.Query, error) {
	parserAST := ast.Copy(program)
	readers := []*querygen.Reader{{}}
	readers[0].Readers = []zio.Reader{r}
	trunk := ast.Trunk{
		Kind:   "Trunk",
		Source: readers[0],
	}
	from := &ast.From{
		Kind:   "From",
		Trunks: []ast.Trunk{trunk},
	}
	seq, ok := parserAST.(*ast.Sequential)
	if !ok {
		return nil, fmt.Errorf("internal error: AST must begin with a Sequential op: %T", parserAST)
	}
	seq.Prepend(from)
	entry, err := semantic.Analyze(pctx.Context, seq)
	if err != nil {
		return nil, err
	}
	optimizer := optimizer.New(pctx.Context, entry, nil)
	builder := querygen.NewBuilder(pctx, nil, nil, nil)
	return optimizeAndBuild(pctx, optimizer, builder)
}

func (*compiler) NewLakeQuery(pctx *op.Context, program ast.Op, parallelism int, head *lakeparse.Commitish) (*runtime.Query, error) {
	panic("NewLakeQuery called on compiler.anyCompiler")
}

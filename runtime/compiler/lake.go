package compiler

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/semantic"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/pkg/storage"
	zedruntime "github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/runtime/exec/optimizer"
	"github.com/brimdata/zed/runtime/exec/querygen"
	"github.com/brimdata/zed/runtime/op"
)

var Parallelism = runtime.GOMAXPROCS(0) //XXX

type lakeCompiler struct {
	compiler
	engine storage.Engine
	lake   *lake.Root
}

func NewLakeCompiler(r *lake.Root) zedruntime.Compiler {
	// We configure a remote storage engine into the lake compiler so that
	// "from" operators that source http or s3 will work, but stdio and
	// file system accesses will be rejected at open time.
	return &lakeCompiler{
		engine: storage.NewRemoteEngine(),
		lake:   r,
	}
}

func (l *lakeCompiler) NewLakeQuery(pctx *op.Context, program ast.Op, parallelism int, head *lakeparse.Commitish) (*zedruntime.Query, error) {
	parserAST := ast.Copy(program)
	// An AST always begins with a Sequential op with at least one
	// operator.  If the first op is a From or a Parallel whose branches
	// are Sequentials with a leading From, then we presume there is
	// no externally defined input.  Otherwise, we expect two readers
	// to be defined for a Join and one reader for anything else.  When input
	// is expected like this, we set up one or two readers inside of an
	// automatically inserted From.  These readers can be accessed by the
	// caller via runtime.readers.  In most cases, the AST is left
	// with an ast.From at the entry point, and hence a dag.From for the
	// DAG's entry point.
	seq, ok := parserAST.(*ast.Sequential)
	if !ok {
		return nil, fmt.Errorf("internal error: AST must begin with a Sequential op: %T", parserAST)
	}
	if len(seq.Ops) == 0 {
		return nil, errors.New("internal error: AST Sequential op cannot be empty")
	}
	if _, ok := seq.Ops[0].(*ast.From); !ok && !isParallelWithLeadingFroms(seq.Ops[0]) {
		trunk := ast.Trunk{Kind: "Trunk"}
		if head == nil {
			return nil, errors.New("query does not specify source of data: no from and no HEAD")
		}
		// For the lakes, if there is no from operator, then
		// we default to scanning HEAD (without any of the
		// from options).
		trunk.Source = &ast.Pool{
			Kind: "Pool",
			Spec: ast.PoolSpec{Pool: "HEAD"},
		}
		seq.Prepend(&ast.From{
			Kind:   "From",
			Trunks: []ast.Trunk{trunk},
		})
	}
	entry, err := semantic.Analyze(pctx.Context, seq)
	if err != nil {
		return nil, err
	}
	optimizer := optimizer.New(pctx.Context, entry, l.lake)
	if err := optimizer.OptimizeScan(); err != nil {
		return nil, err
	}
	if parallelism == 0 {
		parallelism = Parallelism
	}
	if parallelism > 1 {
		if err := optimizer.Parallelize(parallelism); err != nil {
			return nil, err
		}
	}
	builder := querygen.NewBuilder(pctx, l.engine, l.lake, head)
	outputs, err := builder.Build(optimizer.Entry())
	if err != nil {
		return nil, err
	}
	output := newOutput(pctx, outputs)
	return zedruntime.NewQuery(pctx, output, builder.Meters()), nil
}

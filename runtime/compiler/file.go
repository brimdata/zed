package compiler

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/semantic"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/runtime/exec/optimizer"
	"github.com/brimdata/zed/runtime/exec/querygen"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zio"
)

type fsCompiler struct {
	compiler
	engine storage.Engine
}

func NewFileSystemCompiler(engine storage.Engine) runtime.Compiler {
	return &fsCompiler{engine: engine}
}

func (f *fsCompiler) NewQuery(pctx *op.Context, program ast.Op, inputs []zio.Reader) (*runtime.Query, error) {
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
	var readers []*querygen.Reader
	switch o := seq.Ops[0].(type) {
	case *ast.From:
		// Already have an entry point with From.  Do nothing.
	case *ast.Join:
		readers = []*querygen.Reader{{}, {}}
		if len(inputs) != 2 {
			return nil, errors.New("join operaetor requires two inputs")
		}
		readers[0].Readers = inputs[0:1]
		readers[1].Readers = inputs[1:2]
		trunk0 := ast.Trunk{
			Kind:   "Trunk",
			Source: readers[0],
		}
		trunk1 := ast.Trunk{
			Kind:   "Trunk",
			Source: readers[1],
		}
		from := &ast.From{
			Kind:   "From",
			Trunks: []ast.Trunk{trunk0, trunk1},
		}
		seq.Prepend(from)
	default:
		if !isParallelWithLeadingFroms(o) {
			trunk := ast.Trunk{Kind: "Trunk"}
			readers = []*querygen.Reader{{}}
			trunk.Source = readers[0]
			from := &ast.From{
				Kind:   "From",
				Trunks: []ast.Trunk{trunk},
			}
			seq.Prepend(from)
		}
	}
	if len(inputs) == 0 {
		// If there's no inputs but the DAG wants an input, then
		// flag an error.
		if len(readers) != 0 {
			return nil, errors.New("no input specified: use a command-line file or a Zed source operator")
		}
	} else {
		// If there's a reader but the DAG doesn't want an input,
		// then flag an error.
		// TBD: we could have such a configuration is a composite
		// from command includes a "pass" operator, but we can add this later.
		// See issue #2640.
		if len(readers) == 0 {
			return nil, errors.New("redundant inputs specified: use either command-line files or a Zed source operator")
		}
		if len(readers) != 1 {
			return nil, errors.New("Zed query requires a single input path")
		}
		readers[0].Readers = inputs
	}
	entry, err := semantic.Analyze(pctx.Context, seq)
	if err != nil {
		return nil, err
	}
	optimizer := optimizer.New(pctx.Context, entry, nil)
	builder := querygen.NewBuilder(pctx, f.engine, nil, nil)
	return optimizeAndBuild(pctx, optimizer, builder)
}

func (*fsCompiler) NewLakeQuery(pctx *op.Context, program ast.Op, parallelism int, head *lakeparse.Commitish) (*runtime.Query, error) {
	panic("NewLakeQuery called on compiler.fsCompiler")
}

func optimizeAndBuild(pctx *op.Context, optimizer *optimizer.Optimizer, builder *querygen.Builder) (*runtime.Query, error) {
	// Call optimize to possible push down a filter predicate into the
	// kernel.Reader so that the zng scanner can do boyer-moore.
	if err := optimizer.OptimizeScan(); err != nil {
		return nil, err
	}
	// For an internal reader (like a shaper on intake), we don't do
	// any parallelization right now though this could be potentially
	// beneficial depending on where the bottleneck is for a given shaper.
	// See issue #2641.
	outputs, err := builder.Build(optimizer.Entry())
	if err != nil {
		return nil, err
	}
	output := newOutput(pctx, outputs)
	return runtime.NewQuery(pctx, output, builder.Meters()), nil
}

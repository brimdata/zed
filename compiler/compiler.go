package compiler

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/kernel"
	"github.com/brimdata/zed/compiler/optimizer"
	"github.com/brimdata/zed/compiler/parser"
	"github.com/brimdata/zed/compiler/semantic"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zbuf"
)

type Runtime struct {
	pctx      *op.Context
	builder   *kernel.Builder
	optimizer *optimizer.Optimizer
	consts    []dag.Op
	outputs   []zbuf.Puller
	readers   []*kernel.Reader
	puller    zbuf.Puller
	meter     *meter
}

func New(pctx *op.Context, inAST ast.Proc, adaptor op.DataAdaptor, head *lakeparse.Commitish) (*Runtime, error) {
	parserAST := ast.Copy(inAST)
	// An AST always begins with a Sequential op with at least one
	// operator.  If the first proc is a From or a Parallel whose branches
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
		return nil, fmt.Errorf("internal error: AST must begin with a Sequential proc: %T", parserAST)
	}
	if len(seq.Procs) == 0 {
		return nil, errors.New("internal error: AST Sequential proc cannot be empty")
	}
	var readers []*kernel.Reader
	var from *ast.From
	switch proc := seq.Procs[0].(type) {
	case *ast.From:
		// Already have an entry point with From.  Do nothing.
	case *ast.Join:
		readers = []*kernel.Reader{{}, {}}
		trunk0 := ast.Trunk{
			Kind:   "Trunk",
			Source: readers[0],
		}
		trunk1 := ast.Trunk{
			Kind:   "Trunk",
			Source: readers[1],
		}
		from = &ast.From{
			Kind:   "From",
			Trunks: []ast.Trunk{trunk0, trunk1},
		}
	default:
		trunk := ast.Trunk{Kind: "Trunk"}
		if head != nil {
			// For the lakes, if there is no from operator, then
			// we default to scanning HEAD (without any of the
			// from options).
			trunk.Source = &ast.Pool{
				Kind: "Pool",
				Spec: ast.PoolSpec{Pool: "HEAD"},
			}
		} else {
			readers = []*kernel.Reader{{}}
			trunk.Source = readers[0]
		}
		from = &ast.From{
			Kind:   "From",
			Trunks: []ast.Trunk{trunk},
		}
		if isParallelWithLeadingFroms(proc) {
			from = nil
			readers = nil
		}
	}
	if from != nil {
		seq.Prepend(from)
	}
	entry, err := semantic.Analyze(pctx.Context, seq, adaptor, head)
	if err != nil {
		return nil, err
	}
	return &Runtime{
		pctx:      pctx,
		builder:   kernel.NewBuilder(pctx, adaptor),
		optimizer: optimizer.New(pctx.Context, entry, adaptor),
		readers:   readers,
	}, nil
}

func isParallelWithLeadingFroms(p ast.Proc) bool {
	par, ok := p.(*ast.Parallel)
	if !ok {
		return false
	}
	for _, p := range par.Procs {
		if !isSequentialWithLeadingFrom(p) {
			return false
		}
	}
	return true
}

func isSequentialWithLeadingFrom(p ast.Proc) bool {
	seq, ok := p.(*ast.Sequential)
	if !ok && len(seq.Procs) == 0 {
		return false
	}
	_, ok = seq.Procs[0].(*ast.From)
	return ok
}

func (r *Runtime) Context() *op.Context {
	return r.pctx
}

func (r *Runtime) Outputs() []zbuf.Puller {
	return r.outputs
}

func (r *Runtime) Entry() dag.Op {
	//XXX need to prepend consts depending on context
	return r.optimizer.Entry()
}

func (r *Runtime) Meter() zbuf.Meter {
	return r.meter
}

// This must be called before the zbuf.Filter interface will work.
func (r *Runtime) Optimize() error {
	return r.optimizer.OptimizeScan()
}

func (r *Runtime) Parallelize(n int) error {
	return r.optimizer.Parallelize(n)
}

// ParseProc() is an entry point for use from external go code,
// mostly just a wrapper around Parse() that casts the return value.
func ParseProc(src string, filenames ...string) (ast.Proc, error) {
	parsed, err := parser.ParseZed(filenames, src)
	if err != nil {
		return nil, err
	}
	return ast.UnpackMapAsProc(parsed)
}

func ParseExpression(expr string) (ast.Expr, error) {
	m, err := parser.ParseZedByRule("Expr", expr)
	if err != nil {
		return nil, err
	}
	return ast.UnpackMapAsExpr(m)
}

// MustParseProc is functionally the same as ParseProc but panics if an error
// is encountered.
func MustParseProc(query string) ast.Proc {
	proc, err := ParseProc(query)
	if err != nil {
		panic(err)
	}
	return proc
}

func (r *Runtime) Builder() *kernel.Builder {
	return r.builder
}

func (r *Runtime) Build() error {
	outputs, err := r.builder.Build(r.optimizer.Entry())
	if err != nil {
		return err
	}
	r.outputs = outputs
	r.meter = &meter{r.builder.Meters()}
	return nil
}

func (r *Runtime) Puller() zbuf.Puller {
	if r.puller == nil {
		switch outputs := r.Outputs(); len(outputs) {
		case 0:
			return nil
		case 1:
			r.puller = op.NewCatcher(op.NewSingle(outputs[0]))
		default:
			r.puller = op.NewMux(r.pctx, outputs)
		}
	}
	return r.puller
}

func CompileAssignments(zctx *zed.Context, dsts field.List, srcs field.List) (field.List, []expr.Evaluator) {
	return kernel.CompileAssignments(zctx, dsts, srcs)
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

package compiler

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/kernel"
	"github.com/brimdata/zed/compiler/optimizer"
	"github.com/brimdata/zed/compiler/parser"
	"github.com/brimdata/zed/compiler/semantic"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/proc/combine"
	"github.com/brimdata/zed/zbuf"
)

type Runtime struct {
	pctx      *proc.Context
	builder   *kernel.Builder
	optimizer *optimizer.Optimizer
	consts    []dag.Op
	outputs   []proc.Interface
	readers   []*kernel.Reader
}

func New(pctx *proc.Context, inAST ast.Proc, adaptor proc.DataAdaptor) (*Runtime, error) {
	parserAST := ast.Copy(inAST)
	// An AST always begins with a Sequential proc with at least one
	// proc.  If the first proc is a From, then we presume there is no
	// externally defined input.  Otherwise, we expect two readers
	// to be defined for a Join and one reader for anything else.  When input
	// is expected like this, we set up one or two readers inside of an
	// autamitcally inserted From.  These readers can be accessed by the
	// caller via runtime.readers.  In all cases, the AST is left
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
	case *ast.Parallel:
		readers = make([]*kernel.Reader, len(proc.Procs))
		trunks := make([]ast.Trunk, len(proc.Procs))
		for k := range proc.Procs {
			readers[k] = &kernel.Reader{}
			trunks[k] = ast.Trunk{
				Kind:   "Trunk",
				Source: readers[k],
			}
		}
		from = &ast.From{
			Kind:   "From",
			Trunks: trunks,
		}
	default:
		readers = []*kernel.Reader{{}}
		trunk := ast.Trunk{
			Kind:   "Trunk",
			Source: readers[0],
		}
		from = &ast.From{
			Kind:   "From",
			Trunks: []ast.Trunk{trunk},
		}
	}
	if from != nil {
		seq.Prepend(from)
	}
	entry, consts, err := semantic.Analyze(pctx.Context, seq, adaptor)
	if err != nil {
		return nil, err
	}
	opt, err := optimizer.New(pctx.Context, entry, adaptor)
	if err != nil {
		return nil, err
	}
	builder := kernel.NewBuilder(pctx, adaptor)
	if err := builder.LoadConsts(consts); err != nil {
		return nil, err
	}
	return &Runtime{
		pctx:      pctx,
		builder:   builder,
		optimizer: opt,
		consts:    consts,
		readers:   readers,
	}, nil
}

func (r *Runtime) Context() *proc.Context {
	return r.pctx
}

func (r *Runtime) Outputs() []proc.Interface {
	return r.outputs
}

func (r *Runtime) Entry() dag.Op {
	//XXX need to prepend consts depending on context
	return r.optimizer.Entry()
}

func (r *Runtime) Statser() zbuf.Statser {
	return newStatser(r.builder.Schedulers())
}

// This must be called before the zbuf.Filter interface will work.
func (r *Runtime) Optimize() error {
	r.optimizer.OptimizeScan()
	return nil
}

func (r *Runtime) Parallelize(n int) error {
	return r.optimizer.Parallelize(n)
}

// ParseProc() is an entry point for use from external go code,
// mostly just a wrapper around Parse() that casts the return value.
func ParseProc(z string) (ast.Proc, error) {
	parsed, err := parser.ParseZ(z)
	if err != nil {
		return nil, err
	}
	return ast.UnpackMapAsProc(parsed)
}

func ParseExpression(expr string) (ast.Expr, error) {
	m, err := parser.ParseZByRule("Expr", expr)
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
	return r.BuildCustom(nil)
}

func (r *Runtime) BuildCustom(custom kernel.Hook) error {
	r.builder.Custom = custom //XXX
	outputs, err := r.builder.Build(r.optimizer.Entry())
	if err != nil {
		return err
	}
	r.outputs = outputs
	return nil
}

func (r *Runtime) AsPuller() zbuf.Puller {
	outputs := r.Outputs()
	switch len(outputs) {
	case 0:
		return nil
	case 1:
		return outputs[0]
	default:
		return combine.New(r.pctx, outputs)
	}
}

func CompileAssignments(dsts []field.Static, srcs []field.Static) ([]field.Static, []expr.Evaluator) {
	return kernel.CompileAssignments(dsts, srcs)
}

type statser struct {
	schedulers []proc.Scheduler
}

func newStatser(schedulers []proc.Scheduler) *statser {
	return &statser{schedulers}
}

func (s *statser) Stats() zbuf.ScannerStats {
	var out zbuf.ScannerStats
	for _, sched := range s.schedulers {
		out.Add(sched.Stats())
	}
	return out
}

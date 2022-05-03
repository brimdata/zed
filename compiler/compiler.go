package compiler

import (
	"context"
	"errors"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/kernel"
	"github.com/brimdata/zed/compiler/optimizer"
	"github.com/brimdata/zed/compiler/parser"
	"github.com/brimdata/zed/compiler/semantic"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/order"
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

func New(pctx *op.Context, inAST ast.Op, adaptor op.DataAdaptor, head *lakeparse.Commitish) (*Runtime, error) {
	parserAST := ast.Copy(inAST)
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
	var readers []*kernel.Reader
	var from *ast.From
	switch o := seq.Ops[0].(type) {
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
		if isParallelWithLeadingFroms(o) {
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

func isParallelWithLeadingFroms(o ast.Op) bool {
	par, ok := o.(*ast.Parallel)
	if !ok {
		return false
	}
	for _, o := range par.Ops {
		if !isSequentialWithLeadingFrom(o) {
			return false
		}
	}
	return true
}

func isSequentialWithLeadingFrom(o ast.Op) bool {
	seq, ok := o.(*ast.Sequential)
	if !ok && len(seq.Ops) == 0 {
		return false
	}
	_, ok = seq.Ops[0].(*ast.From)
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

// ParseOp concatenates the source files in filenames followed by src and parses
// the resulting program.
func ParseOp(src string, filenames ...string) (ast.Op, error) {
	parsed, err := parser.ParseZed(filenames, src)
	if err != nil {
		return nil, err
	}
	return ast.UnpackMapAsOp(parsed)
}

// MustParseOp is like ParseOp but panics if an error is encountered.
func MustParseOp(query string) ast.Op {
	o, err := ParseOp(query)
	if err != nil {
		panic(err)
	}
	return o
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

func ParseRangeExpr(zctx *zed.Context, src string, layout order.Layout) (*zed.Value, string, error) {
	o, err := ParseOp(src)
	if err != nil {
		return nil, "", err
	}
	d, err := semantic.Analyze(context.Background(), o.(*ast.Sequential), nil, nil)
	if err != nil {
		return nil, "", err
	}
	if len(d.Ops) != 1 {
		return nil, "", errors.New("range expression should only have one operator")
	}
	f, ok := d.Ops[0].(*dag.Filter)
	if !ok {
		return nil, "", errors.New("range expression should be a filter")
	}
	be, ok := f.Expr.(*dag.BinaryExpr)
	if !ok {
		return nil, "", errors.New("must be a simple compare expression")
	}
	switch be.Op {
	case "<=", "<", ">=", ">":
	default:
		return nil, "", fmt.Errorf("unsupported operator: %q", be.Op)
	}
	this, ok := be.LHS.(*dag.This)
	if !ok {
		return nil, "", fmt.Errorf("left hand side must be a path")
	}
	path := field.Path(this.Path)
	if !layout.Keys.Equal(field.List{path}) {
		return nil, "", fmt.Errorf("field %q does not match pool key %q", path, layout.Keys)
	}
	val, err := kernel.EvalAtCompileTime(zctx, be.RHS)
	if err != nil {
		return nil, "", err
	}
	return val, be.Op, nil
}

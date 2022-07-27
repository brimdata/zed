package compiler

import (
	"context"
	"errors"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/parser"
	"github.com/brimdata/zed/compiler/semantic"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/runtime/exec/optimizer"
	"github.com/brimdata/zed/runtime/exec/querygen"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zbuf"
)

type Job struct {
	pctx      *op.Context
	builder   *querygen.Builder
	optimizer *optimizer.Optimizer
	consts    []dag.Op
	outputs   []zbuf.Puller
	readers   []*querygen.Reader
	puller    zbuf.Puller
}

func NewJob(pctx *op.Context, inAST ast.Op, src *querygen.Source, head *lakeparse.Commitish) (*Job, error) {
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
	var readers []*querygen.Reader
	var from *ast.From
	switch o := seq.Ops[0].(type) {
	case *ast.From:
		// Already have an entry point with From.  Do nothing.
	case *ast.Join:
		readers = []*querygen.Reader{{}, {}}
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
			readers = []*querygen.Reader{{}}
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
	entry, err := semantic.Analyze(pctx.Context, seq)
	if err != nil {
		return nil, err
	}
	return &Job{
		pctx:      pctx,
		builder:   querygen.NewBuilder(pctx, src, head),
		optimizer: optimizer.New(pctx.Context, entry, src),
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

func (j *Job) Entry() dag.Op {
	//XXX need to prepend consts depending on context
	return j.optimizer.Entry()
}

// This must be called before the zbuf.Filter interface will work.
func (j *Job) Optimize() error {
	return j.optimizer.OptimizeScan()
}

func (j *Job) Parallelize(n int) error {
	return j.optimizer.Parallelize(n)
}

func Parse(src string, filenames ...string) (ast.Op, error) {
	parsed, err := parser.ParseZed(filenames, src)
	if err != nil {
		return nil, err
	}
	return ast.UnpackMapAsOp(parsed)
}

// MustParse is like Parse but panics if an error is encountered.
func MustParse(query string) ast.Op {
	o, err := (*anyCompiler)(nil).Parse(query)
	if err != nil {
		panic(err)
	}
	return o
}

func (j *Job) Builder() *querygen.Builder {
	return j.builder
}

func (j *Job) Build() error {
	outputs, err := j.builder.Build(j.optimizer.Entry())
	if err != nil {
		return err
	}
	j.outputs = outputs
	return nil
}

func (j *Job) Puller() zbuf.Puller {
	if j.puller == nil {
		switch outputs := j.outputs; len(outputs) {
		case 0:
			return nil
		case 1:
			j.puller = op.NewCatcher(op.NewSingle(outputs[0]))
		default:
			j.puller = op.NewMux(j.pctx, outputs)
		}
	}
	return j.puller
}

type anyCompiler struct{}

// Parse concatenates the source files in filenames followed by src and parses
// the resulting program.
func (*anyCompiler) Parse(src string, filenames ...string) (ast.Op, error) {
	return Parse(src, filenames...)
}

func (a *anyCompiler) ParseRangeExpr(zctx *zed.Context, src string, layout order.Layout) (*zed.Value, string, error) {
	o, err := a.Parse(src)
	if err != nil {
		return nil, "", err
	}
	d, err := semantic.Analyze(context.Background(), o.(*ast.Sequential))
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
	val, err := querygen.EvalAtCompileTime(zctx, be.RHS) //XXX evalAtCompileTime
	if err != nil {
		return nil, "", err
	}
	return val, be.Op, nil
}

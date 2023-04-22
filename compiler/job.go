package compiler

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/data"
	"github.com/brimdata/zed/compiler/kernel"
	"github.com/brimdata/zed/compiler/optimizer"
	"github.com/brimdata/zed/compiler/parser"
	"github.com/brimdata/zed/compiler/semantic"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zbuf"
)

type Job struct {
	octx      *op.Context
	builder   *kernel.Builder
	optimizer *optimizer.Optimizer
	outputs   []zbuf.Puller
	readers   []*kernel.Reader
	puller    zbuf.Puller
}

func NewJob(octx *op.Context, inAST ast.Op, src *data.Source, head *lakeparse.Commitish) (*Job, error) {
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
	scope, ok := parserAST.(*ast.Scope)
	if !ok {
		return nil, fmt.Errorf("internal error: AST must begin with a Scope op: %T", parserAST)
	}
	seq := scope.Body
	if len(seq.Ops) == 0 {
		return nil, errors.New("internal error: AST Scope op body cannot be empty")
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
				Spec: ast.PoolSpec{
					Pool: &ast.String{
						Kind: "String",
						Text: "HEAD",
					},
				},
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
	entry, err := semantic.Analyze(octx.Context, scope, src, head)
	if err != nil {
		return nil, err
	}
	return &Job{
		octx:      octx,
		builder:   kernel.NewBuilder(octx, src),
		optimizer: optimizer.New(octx.Context, entry, src),
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
	return j.optimizer.Optimize()
}

func (j *Job) OptimizeDeleter(replicas int) error {
	return j.optimizer.OptimizeDeleter(replicas)
}

func (j *Job) Parallelize(n int) error {
	return j.optimizer.Parallelize(n)
}

func ParseNoWrap(src string, filenames ...string) (ast.Op, error) {
	parsed, err := parser.ParseZed(filenames, src)
	if err != nil {
		return nil, err
	}
	return ast.UnpackMapAsOp(parsed)
}

func Parse(src string, filenames ...string) (ast.Op, error) {
	op, err := ParseNoWrap(src, filenames...)
	return wrapScope(op), err
}

func wrapScope(op ast.Op) ast.Op {
	if seq, ok := op.(*ast.Sequential); ok {
		op = &ast.Scope{Kind: "Scope", Body: seq}
	}
	return op
}

// MustParse is like Parse but panics if an error is encountered.
func MustParse(query string) ast.Op {
	o, err := (*anyCompiler)(nil).Parse(query)
	if err != nil {
		panic(err)
	}
	return o
}

func (j *Job) Builder() *kernel.Builder {
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
			j.puller = op.NewMux(j.octx, outputs)
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

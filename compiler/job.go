package compiler

import (
	"errors"

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
	reader    *kernel.Reader
	puller    zbuf.Puller
	entry     dag.Seq
}

func NewJob(octx *op.Context, in ast.Seq, src *data.Source, head *lakeparse.Commitish) (*Job, error) {
	seq := ast.CopySeq(in)
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
	if len(seq) == 0 {
		return nil, errors.New("internal error: AST seq cannot be empty")
	}
	from, reader, err := buildFrom(seq[0], head)
	if err != nil {
		return nil, err
	}
	if from != nil {
		seq.Prepend(from)
	}
	entry, err := semantic.Analyze(octx.Context, seq, src, head)
	if err != nil {
		return nil, err
	}
	return &Job{
		octx:      octx,
		builder:   kernel.NewBuilder(octx, src),
		optimizer: optimizer.New(octx.Context, src),
		reader:    reader,
		entry:     entry,
	}, nil
}

func buildFrom(op ast.Op, head *lakeparse.Commitish) (*ast.From, *kernel.Reader, error) {
	var from *ast.From
	switch op := op.(type) {
	case *ast.From:
		// Already have an entry point with From.  Do nothing.
		return nil, nil, nil
	case *ast.Scope:
		if len(op.Body) == 0 {
			return nil, nil, errors.New("internal error: scope op has empty body")
		}
		return buildFrom(op.Body[0], head)
	default:
		var readers *kernel.Reader
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
			readers = &kernel.Reader{}
			trunk.Source = readers
		}
		from = &ast.From{
			Kind:   "From",
			Trunks: []ast.Trunk{trunk},
		}
		// XXX why not move this above? or
		if isParallelWithLeadingFroms(op) {
			from = nil
			readers = nil
		}
		return from, readers, nil
	}
}

func isParallelWithLeadingFroms(o ast.Op) bool {
	par, ok := o.(*ast.Parallel)
	if !ok {
		return false
	}
	for _, seq := range par.Paths {
		if !hasLeadingFrom(seq) {
			return false
		}
	}
	return true
}

func hasLeadingFrom(seq ast.Seq) bool {
	if len(seq) == 0 {
		return false
	}
	_, ok := seq[0].(*ast.From)
	return ok
}

func (j *Job) Entry() dag.Seq {
	//XXX need to prepend consts depending on context
	return j.entry
}

// This must be called before the zbuf.Filter interface will work.
func (j *Job) Optimize() error {
	var err error
	j.entry, err = j.optimizer.Optimize(j.entry)
	return err
}

func (j *Job) OptimizeDeleter(replicas int) error {
	var err error
	j.entry, err = j.optimizer.OptimizeDeleter(j.entry, replicas)
	return err
}

func (j *Job) Parallelize(n int) error {
	var err error
	j.entry, err = j.optimizer.Parallelize(j.entry, n)
	return err
}

func Parse(src string, filenames ...string) (ast.Seq, error) {
	parsed, err := parser.ParseZed(filenames, src)
	if err != nil {
		return nil, err
	}
	return ast.UnmarshalObject(parsed)
}

// MustParse is like Parse but panics if an error is encountered.
func MustParse(query string) ast.Seq {
	seq, err := (*anyCompiler)(nil).Parse(query)
	if err != nil {
		panic(err)
	}
	return seq
}

func (j *Job) Builder() *kernel.Builder {
	return j.builder
}

func (j *Job) Build() error {
	outputs, err := j.builder.Build(j.entry)
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
func (*anyCompiler) Parse(src string, filenames ...string) (ast.Seq, error) {
	return Parse(src, filenames...)
}

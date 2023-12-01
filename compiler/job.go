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
	"github.com/brimdata/zed/zio"
)

type Job struct {
	octx      *op.Context
	builder   *kernel.Builder
	optimizer *optimizer.Optimizer
	outputs   []zbuf.Puller
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
	entry, err := semantic.AnalyzeAddSource(octx.Context, seq, src, head)
	if err != nil {
		return nil, err
	}
	return &Job{
		octx:      octx,
		builder:   kernel.NewBuilder(octx, src),
		optimizer: optimizer.New(octx.Context, src),
		entry:     entry,
	}, nil
}

func (j *Job) DefaultScan() (*dag.DefaultScan, bool) {
	scan, ok := j.entry[0].(*dag.DefaultScan)
	return scan, ok
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

// Build builds a flowgraph for j.  If the flowgraph expects an input stream
// from the runtime (because its DAG contains a dag.DefaultSource), Build
// constructs it from readers.
func (j *Job) Build(readers ...zio.Reader) error {
	outputs, err := j.builder.Build(j.entry, readers...)
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

package explode

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zcode"
)

// A an explode Proc is a proc that, given an input record and a
// zng type T, outputs one record for each field of the input record of
// type T. It is useful for type-based indexing.
type Proc struct {
	parent  proc.Interface
	builder zed.Builder
	typ     zed.Type
	args    []expr.Evaluator
}

// New creates a exploder for type typ, where the
// output records' single column is named name.
func New(zctx *zed.Context, parent proc.Interface, args []expr.Evaluator, typ zed.Type, name string) (proc.Interface, error) {
	cols := []zed.Column{{Name: name, Type: typ}}
	rectyp := zctx.MustLookupTypeRecord(cols)
	builder := zed.NewBuilder(rectyp)
	return &Proc{
		parent:  parent,
		builder: *builder,
		typ:     typ,
		args:    args,
	}, nil
}

func (p *Proc) Pull() (zbuf.Batch, error) {
	for {
		batch, err := p.parent.Pull()
		if proc.EOS(batch, err) {
			return nil, err
		}
		ctx := batch.Context()
		vals := batch.Values()
		out := make([]zed.Value, 0, len(vals))
		for i := range vals {
			for _, arg := range p.args {
				val := arg.Eval(ctx, &vals[i])
				if val.IsError() {
					if val != zed.Missing {
						out = append(out, *val.Copy())
					}
					continue
				}
				zed.Walk(val.Type, val.Bytes, func(typ zed.Type, body zcode.Bytes) error {
					if typ == p.typ && body != nil {
						out = append(out, *p.builder.Build(body))
						return zed.SkipContainer
					}
					return nil
				})
			}
		}
		batch.Unref()
		if len(out) > 0 {
			return zbuf.NewArray(out), nil
		}
	}
}

func (p *Proc) Done() {
	p.parent.Done()
}

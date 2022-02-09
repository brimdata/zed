package explode

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zcode"
)

// A an explode Proc is a proc that, given an input record and a
// zng type T, outputs one record for each field of the input record of
// type T. It is useful for type-based indexing.
type Proc struct {
	parent  zbuf.Puller
	outType zed.Type
	typ     zed.Type
	args    []expr.Evaluator
}

// New creates a exploder for type typ, where the
// output records' single column is named name.
func New(zctx *zed.Context, parent zbuf.Puller, args []expr.Evaluator, typ zed.Type, name string) (zbuf.Puller, error) {
	return &Proc{
		parent:  parent,
		outType: zctx.MustLookupTypeRecord([]zed.Column{{Name: name, Type: typ}}),
		typ:     typ,
		args:    args,
	}, nil
}

func (p *Proc) Pull(done bool) (zbuf.Batch, error) {
	for {
		batch, err := p.parent.Pull(done)
		if batch == nil || err != nil {
			return nil, err
		}
		vals := batch.Values()
		out := make([]zed.Value, 0, len(vals))
		for i := range vals {
			for _, arg := range p.args {
				val := arg.Eval(batch, &vals[i])
				if val.IsError() {
					if !val.IsMissing() {
						out = append(out, *val.Copy())
					}
					continue
				}
				zed.Walk(val.Type, val.Bytes, func(typ zed.Type, body zcode.Bytes) error {
					if typ == p.typ && body != nil {
						bytes := zcode.Append(nil, body)
						out = append(out, *zed.NewValue(p.outType, bytes))
						return zed.SkipContainer
					}
					return nil
				})
			}
		}
		if len(out) > 0 {
			defer batch.Unref()
			return zbuf.NewBatch(batch, out), nil
		}
		batch.Unref()
	}
}

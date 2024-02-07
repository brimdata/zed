package explode

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/runtime/sam/expr"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zcode"
)

// A an explode Proc is a proc that, given an input record and a
// zng type T, outputs one record for each field of the input record of
// type T. It is useful for type-based indexing.
type Op struct {
	parent  zbuf.Puller
	rctx    *runtime.Context
	outType zed.Type
	typ     zed.Type
	args    []expr.Evaluator
}

// New creates a exploder for type typ, where the
// output records' single field is named name.
func New(rctx *runtime.Context, parent zbuf.Puller, args []expr.Evaluator, typ zed.Type, name string) (zbuf.Puller, error) {
	return &Op{
		parent:  parent,
		rctx:    rctx,
		outType: rctx.Zctx.MustLookupTypeRecord([]zed.Field{{Name: name, Type: typ}}),
		typ:     typ,
		args:    args,
	}, nil
}

func (o *Op) Pull(done bool) (zbuf.Batch, error) {
	arena := zed.NewArena(o.rctx.Zctx)
	for {
		batch, err := o.parent.Pull(done)
		if batch == nil || err != nil {
			return nil, err
		}
		ectx := expr.NewContextWithVars(arena, batch.Vars())
		vals := batch.Values()
		out := make([]zed.Value, 0, len(vals))
		for _, val := range vals {
			for _, arg := range o.args {
				val := arg.Eval(ectx, val)
				if val.IsError() {
					if !val.IsMissing() {
						out = append(out, val.Copy())
					}
					continue
				}
				zed.Walk(val.Type(), val.Bytes(), func(typ zed.Type, body zcode.Bytes) error {
					if typ == o.typ && body != nil {
						bytes := zcode.Append(nil, body)
						out = append(out, arena.NewValue(o.outType, bytes))
						return zed.SkipContainer
					}
					return nil
				})
			}
		}
		if len(out) > 0 {
			defer arena.Unref()
			defer batch.Unref()
			return zbuf.NewBatch(arena, out, batch, batch.Vars()), nil
		}
		arena.Unref()
		batch.Unref()
	}
}

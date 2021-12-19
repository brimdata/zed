package traverse

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zcode"
)

type Over struct {
	exprs  []expr.Evaluator
	parent proc.Interface
	batch  zbuf.Batch
	vals   []zed.Value
	eof    bool
}

func NewOver(parent proc.Interface, exprs []expr.Evaluator) *Over {
	return &Over{
		exprs:  exprs,
		parent: parent,
	}
}

func (o *Over) Pull() (zbuf.Batch, error) {
	if len(o.vals) == 0 {
		batch, err := o.parent.Pull()
		if batch == nil || err != nil {
			return batch, err
		}
		o.eof = false
		o.batch = batch
		o.vals = batch.Values()
	}
	if o.eof {
		o.eof = false
		return nil, nil
	}
	o.eof = true
	out, err := o.over(&o.vals[0], o.batch.Scope())
	o.vals = o.vals[1:]
	if len(o.vals) == 0 {
		o.batch.Unref()
	}
	return out, err
}

// Done is currently ignored as the model here as each downstream batch should be
// handled indepedently.  We need a way to scope flowgraphs so the done protocol can
// be propagated on an outer scope but not on the inner scope.
func (o *Over) Done() {}

func (o *Over) over(this *zed.Value, scope *expr.Scope) (*zbuf.Array, error) {
	var vals []zed.Value
	for _, e := range o.exprs {
		val := e.Eval(this, scope)
		// Propagate errors but skip missing values.
		if val != zed.Missing {
			var err error
			if vals, err = appendOver(vals, *val); err != nil {
				return nil, err
			}
		}
	}
	return zbuf.NewArray(vals), nil

}

func appendOver(vals []zed.Value, zv zed.Value) ([]zed.Value, error) {
	if zed.IsPrimitiveType(zv.Type) {
		return append(vals, zv), nil
	}
	typ := zed.InnerType(zv.Type)
	if typ == nil {
		// XXX Issue #3324: need to support records and maps.
		return vals, nil
	}
	iter := zcode.Iter(zv.Bytes)
	for {
		b, _, err := iter.Next()
		if b == nil {
			return vals, nil
		}
		if err != nil {
			return nil, err
		}
		//XXX zbuf.Array should be zed.Value not pointer?!
		// also, we need to copy the value since we the caller
		// wants to unref the input batch.
		// In a future batch implementation, we should be able to
		// refcnt the underlying slice buffers and share the buffers
		// across different batches.
		bc := make([]byte, len(b))
		copy(bc, b)
		vals = append(vals, zed.Value{typ, bc})
	}
}

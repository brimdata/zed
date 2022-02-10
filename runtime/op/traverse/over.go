package traverse

import (
	"context"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zcode"
)

type Over struct {
	parent zbuf.Puller
	exprs  []expr.Evaluator
	outer  []zed.Value
	batch  zbuf.Batch
	enter  *Enter
	zctx   *zed.Context
}

func NewOver(pctx *op.Context, parent zbuf.Puller, exprs []expr.Evaluator) *Over {
	return &Over{
		parent: parent,
		exprs:  exprs,
		zctx:   pctx.Zctx,
	}
}

func (o *Over) AddScope(ctx context.Context, names []string, exprs []expr.Evaluator) *Scope {
	scope := newScope(ctx, o, names, exprs)
	o.enter = scope.enter
	return scope
}

func (o *Over) Pull(done bool) (zbuf.Batch, error) {
	if done {
		o.outer = nil
		return o.parent.Pull(true)
	}
	if len(o.outer) == 0 {
		batch, err := o.parent.Pull(false)
		if batch == nil || err != nil {
			return nil, err
		}
		o.batch = batch
		o.outer = batch.Values()
	}
	this := &o.outer[0]
	o.outer = o.outer[1:]
	ectx := o.batch
	if o.enter != nil {
		ectx = o.enter.addLocals(ectx, this)
	}
	innerBatch := o.over(ectx, this)
	if len(o.outer) == 0 {
		o.batch.Unref()
	}
	return innerBatch, nil
}

func (o *Over) over(batch zbuf.Batch, this *zed.Value) zbuf.Batch {
	// Copy the vars into a new scope since downstream, nested subgraphs
	// can have concurrent operators.  We can optimize these copies out
	// later depending on the nested subgraph.
	var vals []zed.Value
	for _, e := range o.exprs {
		val := e.Eval(batch, this)
		// Propagate errors but skip missing values.
		if !val.IsMissing() {
			vals = appendOver(o.zctx, vals, *val)
		}
	}
	return zbuf.NewBatch(batch, vals)
}

func appendOver(zctx *zed.Context, vals []zed.Value, zv zed.Value) []zed.Value {
	zv = *expr.ValueUnder(&zv)
	switch typ := zed.TypeUnder(zv.Type).(type) {
	case *zed.TypeArray, *zed.TypeSet:
		typ = zed.InnerType(typ)
		for it := zv.Bytes.Iter(); !it.Done(); {
			// XXX when we do proper expr.Context, we can allocate
			// this copy through the batch.
			val := expr.ValueUnder(zed.NewValue(typ, it.Next()))
			vals = append(vals, *val.Copy())
		}
		return vals
	case *zed.TypeMap:
		rtyp := zctx.MustLookupTypeRecord([]zed.Column{
			zed.NewColumn("key", typ.KeyType),
			zed.NewColumn("value", typ.ValType),
		})
		for it := zv.Bytes.Iter(); !it.Done(); {
			bytes := zcode.Append(zcode.Append(nil, it.Next()), it.Next())
			vals = append(vals, *zed.NewValue(rtyp, bytes))
		}
		return vals
	case *zed.TypeRecord:
		builder := zcode.NewBuilder()
		columns := typ.Columns
		for i, it := 0, zv.Bytes.Iter(); !it.Done(); i++ {
			builder.Reset()
			col := columns[i]
			typ := zctx.MustLookupTypeRecord([]zed.Column{
				{Name: "key", Type: zed.TypeString},
				{Name: "value", Type: col.Type},
			})
			builder.Append(zed.EncodeString(col.Name))
			builder.Append(it.Next())
			vals = append(vals, *zed.NewValue(typ, builder.Bytes()).Copy())

		}
		return vals
	default:
		return append(vals, zv)
	}
}

package traverse

import (
	"context"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/runtime/sam/expr"
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

func NewOver(rctx *runtime.Context, parent zbuf.Puller, exprs []expr.Evaluator) *Over {
	return &Over{
		parent: parent,
		exprs:  exprs,
		zctx:   rctx.Zctx,
	}
}

func (o *Over) AddScope(ctx context.Context, names []string, exprs []expr.Evaluator) *Scope {
	scope := newScope(ctx, o.zctx, o, names, exprs)
	o.enter = scope.enter
	return scope
}

func (o *Over) Pull(done bool) (zbuf.Batch, error) {
	if done {
		o.outer = nil
		return o.parent.Pull(true)
	}
	for {
		if len(o.outer) == 0 {
			batch, err := o.parent.Pull(false)
			if batch == nil || err != nil {
				return nil, err
			}
			o.batch = batch
			o.outer = batch.Values()
		}
		this := o.outer[0]
		o.outer = o.outer[1:]
		batch := o.batch
		if o.enter != nil {
			batch = o.enter.addLocals(batch, this)
		}
		innerBatch := o.over(batch, this)
		if len(o.outer) == 0 {
			o.batch.Unref()
		}
		if innerBatch != nil {
			return innerBatch, nil
		}
	}
}

func (o *Over) over(batch zbuf.Batch, this zed.Value) zbuf.Batch {
	arena := zed.NewArena()
	defer arena.Unref()
	ectx := expr.NewContextWithVars(arena, batch.Vars())
	// Copy the vars into a new scope since downstream, nested subgraphs
	// can have concurrent operators.  We can optimize these copies out
	// later depending on the nested subgraph.
	var vals []zed.Value
	for _, e := range o.exprs {
		val := e.Eval(ectx, this)
		// Propagate errors but skip missing values.
		if !val.IsMissing() {
			vals = appendOver(o.zctx, arena, vals, val)
		}
	}
	if len(vals) == 0 {
		return nil
	}
	return zbuf.NewBatch(arena, vals, batch, batch.Vars())
}

func appendOver(zctx *zed.Context, arena *zed.Arena, vals []zed.Value, val zed.Value) []zed.Value {
	val = val.Under(arena)
	switch typ := zed.TypeUnder(val.Type()).(type) {
	case *zed.TypeArray, *zed.TypeSet:
		typ = zed.InnerType(typ)
		for it := val.Bytes().Iter(); !it.Done(); {
			val := arena.New(typ, it.Next()).Under(arena)
			vals = append(vals, val)
		}
		return vals
	case *zed.TypeMap:
		rtyp := zctx.MustLookupTypeRecord([]zed.Field{
			zed.NewField("key", typ.KeyType),
			zed.NewField("value", typ.ValType),
		})
		for it := val.Bytes().Iter(); !it.Done(); {
			bytes := zcode.Append(zcode.Append(nil, it.Next()), it.Next())
			vals = append(vals, arena.New(rtyp, bytes))
		}
		return vals
	case *zed.TypeRecord:
		builder := zcode.NewBuilder()
		for i, it := 0, val.Bytes().Iter(); !it.Done(); i++ {
			builder.Reset()
			field := typ.Fields[i]
			typ := zctx.MustLookupTypeRecord([]zed.Field{
				{Name: "key", Type: zctx.LookupTypeArray(zed.TypeString)},
				{Name: "value", Type: field.Type},
			})
			builder.BeginContainer()
			builder.Append(zed.EncodeString(field.Name))
			builder.EndContainer()
			builder.Append(it.Next())
			vals = append(vals, arena.New(typ, builder.Bytes()))
		}
		return vals
	default:
		return append(vals, val)
	}
}

package traverse

import (
	"context"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zcode"
)

// Expr provides provides glue to run a traversal subquery in expression
// context.  It implements zbuf.Puller so it can serve as the data source
// to the subquery as well as expr.Evalulator so it can be called from an
// expression.  Each time it's Eval method is called, it propagates the value
// to the batch channel to be pulled into the scope.  If there is
// just one result, then the value is returned.  If there are multiple results
// then they are returned in an array create union elements if the type varies.
type Expr struct {
	ctx     context.Context
	zctx    *zed.Context
	batchCh chan zbuf.Batch
	eos     bool

	exit *Exit
	out  []zbuf.Batch
}

var _ expr.Evaluator = (*Expr)(nil)
var _ zbuf.Puller = (*Expr)(nil)

func NewExpr(ctx context.Context, zctx *zed.Context) *Expr {
	return &Expr{
		ctx:     ctx,
		zctx:    zctx,
		batchCh: make(chan zbuf.Batch, 1),
	}
}

func (e *Expr) SetExit(exit *Exit) {
	e.exit = exit
}

func (e *Expr) Eval(ectx expr.Context, this *zed.Value) *zed.Value {
	select {
	case e.batchCh <- zbuf.NewArray([]zed.Value{*this}):
	case <-e.ctx.Done():
		return e.zctx.NewError(e.ctx.Err())
	}
	out := e.out[:0]
	for {
		b, err := e.exit.Pull(false)
		if err != nil {
			panic(err)
		}
		if b == nil {
			e.out = out
			return e.combine(ectx, out)
		}
		out = append(out, b)
	}
}

func (e *Expr) combine(ectx expr.Context, batches []zbuf.Batch) *zed.Value {
	switch len(batches) {
	case 0:
		return zed.Null
	case 1:
		return e.makeArray(ectx, batches[0].Values())
	default:
		var vals []zed.Value
		for _, batch := range batches {
			vals = append(vals, batch.Values()...)
		}
		return e.makeArray(ectx, vals)
	}
}

func (e *Expr) makeArray(ectx expr.Context, vals []zed.Value) *zed.Value {
	if len(vals) == 0 {
		return zed.Null
	}
	typ := vals[0].Type
	if len(vals) == 1 {
		return ectx.NewValue(typ, vals[0].Bytes)
	}
	for _, val := range vals[1:] {
		if typ != val.Type {
			return e.makeUnionArray(ectx, vals)
		}
	}
	var b zcode.Builder
	for _, val := range vals {
		b.Append(val.Bytes)
	}
	return ectx.NewValue(e.zctx.LookupTypeArray(typ), b.Bytes())
}

func (e *Expr) makeUnionArray(ectx expr.Context, vals []zed.Value) *zed.Value {
	types := make(map[zed.Type]struct{})
	for _, val := range vals {
		types[val.Type] = struct{}{}
	}
	utypes := make([]zed.Type, 0, len(types))
	for typ := range types {
		utypes = append(utypes, typ)
	}
	union := e.zctx.LookupTypeUnion(utypes)
	var b zcode.Builder
	for _, val := range vals {
		b.BeginContainer()
		b.Append(zed.EncodeInt(int64(union.Selector(val.Type))))
		b.Append(val.Bytes)
		b.EndContainer()
	}
	return ectx.NewValue(e.zctx.LookupTypeArray(union), b.Bytes())
}

func (e *Expr) Pull(done bool) (zbuf.Batch, error) {
	if e.eos {
		e.eos = false
		return nil, nil
	}
	e.eos = true
	select {
	case batch := <-e.batchCh:
		return batch, nil
	case <-e.ctx.Done():
		return nil, e.ctx.Err()
	}
}

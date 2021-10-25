package expr

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr/agg"
)

type Generator interface {
	Init(*zed.Value)
	Next() (zed.Value, error)
}

type MapMethod struct {
	src    Generator
	dollar zed.Value
	expr   Evaluator
	zv     *zed.Value
}

func NewMapMethod(src Generator) *MapMethod {
	return &MapMethod{src: src}
}

func (m *MapMethod) Ref() *zed.Value {
	return &m.dollar
}

func (m *MapMethod) Set(e Evaluator) {
	m.expr = e
}

func (m *MapMethod) Init(rec *zed.Value) {
	m.zv = rec
	m.src.Init(rec)
}

func (m *MapMethod) Next() (zed.Value, error) {
	zv, err := m.src.Next()
	if err != nil || zv.Type == nil {
		return zv, err
	}
	m.dollar = zv
	return m.expr.Eval(m.zv)
}

type FilterMethod struct {
	src    Generator
	dollar zed.Value
	expr   Evaluator
	zv     *zed.Value
}

func NewFilterMethod(src Generator) *FilterMethod {
	return &FilterMethod{src: src}
}

func (f *FilterMethod) Ref() *zed.Value {
	return &f.dollar
}

func (f *FilterMethod) Set(e Evaluator) {
	f.expr = e
}

func (f *FilterMethod) Init(rec *zed.Value) {
	f.zv = rec
	f.src.Init(rec)
}

func (f *FilterMethod) Next() (zed.Value, error) {
	for {
		zv, err := f.src.Next()
		if err != nil || zv.Type == nil {
			return zv, err
		}
		f.dollar = zv
		b, err := f.expr.Eval(f.zv)
		if err != nil {
			return zed.Value{}, err
		}
		if zed.AliasOf(b.Type) != zed.TypeBool {
			return zed.NewErrorf("not a boolean"), nil
		}
		if zed.IsTrue(b.Bytes) {
			return zv, nil
		}
	}
}

type AggExpr struct {
	zctx   *zed.Context
	newAgg agg.Pattern
	src    Generator
}

func NewAggExpr(zctx *zed.Context, pattern agg.Pattern, src Generator) *AggExpr {
	return &AggExpr{
		zctx:   zctx,
		newAgg: pattern,
		src:    src,
	}
}

func (a *AggExpr) Eval(rec *zed.Value) (zed.Value, error) {
	// XXX This is currently really inefficient while we prototype
	// this machinery.  We used to have a Reset() method on aggregators
	// and we should probably reintroduce that for use here so we
	// don't create a new aggregator for every record. See Issue #2068.
	f := a.newAgg()
	a.src.Init(rec)
	for {
		zv, err := a.src.Next()
		if err != nil {
			if err == zed.ErrMissing {
				continue
			}
			return zed.Value{}, err
		}
		if zv.Type == nil {
			return f.Result(a.zctx)
		}
		if err := f.Consume(zv); err != nil {
			return zed.Value{}, err
		}
	}
}

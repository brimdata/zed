package expr

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
)

type PathElem interface {
	Eval(ectx Context, this *zed.Value) (string, *zed.Value)
}

type Path struct {
	elems []PathElem
	cache field.Path
}

func NewPath(evals []PathElem) *Path {
	return &Path{elems: evals}
}

// Eval returns the path of the lval. If there's an error the returned *zed.Value
// will not be nill.
func (l *Path) Eval(ectx Context, this *zed.Value) (field.Path, *zed.Value) {
	l.cache = l.cache[:0]
	for _, e := range l.elems {
		name, val := e.Eval(ectx, this)
		if val != nil {
			return nil, val
		}
		l.cache = append(l.cache, name)
	}
	return l.cache, nil
}

type StaticPathElem struct {
	Name string
}

func (l *StaticPathElem) Eval(_ Context, _ *zed.Value) (string, *zed.Value) {
	return l.Name, nil
}

type ExprPathElem struct {
	caster Evaluator
	eval   Evaluator
}

func NewPathElemExpr(zctx *zed.Context, e Evaluator) *ExprPathElem {
	return &ExprPathElem{
		eval:   e,
		caster: LookupPrimitiveCaster(zctx, zed.TypeString),
	}
}

func (l *ExprPathElem) Eval(ectx Context, this *zed.Value) (string, *zed.Value) {
	val := l.eval.Eval(ectx, this)
	if val.IsError() {
		return "", val
	}
	if !val.IsString() {
		if val = l.caster.Eval(ectx, val); val.IsError() {
			return "", val
		}
	}
	return val.AsString(), nil
}

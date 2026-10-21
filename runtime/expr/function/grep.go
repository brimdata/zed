package function

import (
	"github.com/superdb/super"
	"github.com/superdb/super/runtime/expr"
	"github.com/superdb/super/vector"
	"golang.org/x/text/unicode/norm"
)

type Grep struct {
	sctx    *super.Context
	expr    expr.Evaluator
	pattern string
}

func (g *Grep) Call(args ...vector.Any) vector.Any {
	patternVec, inputVec := args[0], args[1]
	if patternVec.Type().ID() != super.IDString {
		return vector.NewWrappedError(g.sctx, "grep: pattern argument must be a string", patternVec)
	}
	if inputVec.Len() == 0 {
		return vector.NewFalse(0)
	}
	if vec, ok := expr.Catch(g.sctx, expr.CatchAll, patternVec, "grep"); ok {
		return vec
	}
	if vec, ok := expr.Catch(g.sctx, expr.CatchAll, inputVec, "grep"); ok {
		return vec
	}
	if _, ok := vector.Under(patternVec).(*vector.Const); ok {
		pattern := vector.StringValue(patternVec, 0)
		if g.expr == nil || g.pattern != pattern {
			pattern = norm.NFC.String(pattern)
			g.expr = expr.NewSearchString(g.sctx, pattern, &expr.This{})
			g.pattern = pattern
		}
		return g.expr.Eval(inputVec)
	}
	var index [1]uint32
	out := vector.NewFalse(patternVec.Len())
	for i := range patternVec.Len() {
		pattern := vector.StringValue(patternVec, i)
		pattern = norm.NFC.String(pattern)
		search := expr.NewSearchString(g.sctx, pattern, &expr.This{})
		index[0] = i
		v := search.Eval(vector.Pick(inputVec, index[:]))
		if vector.BoolValue(v, 0) {
			out.Set(i)
		}
	}
	return out
}

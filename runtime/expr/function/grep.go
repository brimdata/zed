package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/expr"
	"golang.org/x/text/unicode/norm"
)

type Grep struct {
	grep    expr.Evaluator
	pattern string
	zctx    *zed.Context
}

func (g *Grep) Call(ectx zed.Allocator, vals []zed.Value) *zed.Value {
	patternVal, inputVal := &vals[0], &vals[1]
	if zed.TypeUnder(patternVal.Type()) != zed.TypeString {
		return g.error(ectx, "pattern argument must be a string", patternVal)
	}
	if p := patternVal.AsString(); g.grep == nil || g.pattern != p {
		g.pattern = p
		term := norm.NFC.Bytes(patternVal.Bytes())
		g.grep = expr.NewSearchString(string(term), nil)
	}
	return g.grep.Eval(wrapAllocator{ectx}, inputVal)
}

// XXX This is gross, any reason function shouldn't just accept an expr.Context?
type wrapAllocator struct {
	zed.Allocator
}

func (wrapAllocator) Vars() []zed.Value { return nil }

func (g *Grep) error(ectx zed.Allocator, msg string, val *zed.Value) *zed.Value {
	msg = "grep(): " + msg
	if val == nil {
		return ectx.CopyValue(*g.zctx.NewErrorf(msg))
	}
	return ectx.CopyValue(*g.zctx.WrapError(msg, val))
}

package function

import (
	"regexp"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#regexp
type Regexp struct {
	builder zcode.Builder
	re      *regexp.Regexp
	restr   string
	typ     zed.Type
	err     error
	zctx    *zed.Context
}

func (r *Regexp) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	if !args[0].IsString() {
		return newErrorf(r.zctx, ctx, "regexp: string required for first arg")
	}
	s := zed.DecodeString(args[0].Bytes())
	if r.restr != s {
		r.restr = s
		r.re, r.err = regexp.Compile(r.restr)
	}
	if r.err != nil {
		return newErrorf(r.zctx, ctx, "regexp: %s", r.err)
	}
	if !args[1].IsString() {
		return newErrorf(r.zctx, ctx, "regexp: string required for second arg")
	}
	r.builder.Reset()
	for _, b := range r.re.FindSubmatch(args[1].Bytes()) {
		r.builder.Append(b)
	}
	if r.typ == nil {
		r.typ = r.zctx.LookupTypeArray(zed.TypeString)
	}
	return ctx.NewValue(r.typ, r.builder.Bytes())
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#regexp_replace
type RegexpReplace struct {
	zctx  *zed.Context
	re    *regexp.Regexp
	restr string
	err   error
}

func (r *RegexpReplace) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	sVal := args[0]
	reVal := args[1]
	newVal := args[2]
	if !sVal.IsString() || !reVal.IsString() || !newVal.IsString() {
		return newErrorf(r.zctx, ctx, "regexp_replace: string values required for all args")
	}
	if sVal.IsNull() {
		return zed.Null
	}
	if reVal.IsNull() || newVal.IsNull() {
		return newErrorf(r.zctx, ctx, "regexp_replace: 2nd and 3rd args cannot be null")
	}
	if re := zed.DecodeString(reVal.Bytes()); r.restr != re {
		r.restr = re
		r.re, r.err = regexp.Compile(re)
	}
	if r.err != nil {
		return newErrorf(r.zctx, ctx, "regexp_replace: %s", r.err)
	}
	return ctx.NewValue(zed.TypeString, r.re.ReplaceAll(sVal.Bytes(), newVal.Bytes()))
}

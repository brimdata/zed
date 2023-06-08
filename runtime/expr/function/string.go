package function

import (
	"strings"
	"unicode/utf8"

	"github.com/agnivade/levenshtein"
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#replace
type Replace struct {
	zctx *zed.Context
}

func (r *Replace) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	sVal := args[0]
	oldVal := args[1]
	newVal := args[2]
	if !sVal.IsString() || !oldVal.IsString() || !newVal.IsString() {
		return newErrorf(r.zctx, ctx, "replace: string arg required")
	}
	if sVal.IsNull() {
		return zed.Null
	}
	if oldVal.IsNull() || newVal.IsNull() {
		return newErrorf(r.zctx, ctx, "replace: an input arg is null")
	}
	s := zed.DecodeString(sVal.Bytes())
	old := zed.DecodeString(oldVal.Bytes())
	new := zed.DecodeString(newVal.Bytes())
	return newString(ctx, strings.ReplaceAll(s, old, new))
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#run_len
type RuneLen struct {
	zctx *zed.Context
}

func (r *RuneLen) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	val := args[0]
	if !val.IsString() {
		return newErrorf(r.zctx, ctx, "rune_len: string arg required")
	}
	if val.IsNull() {
		return ctx.CopyValue(zed.NewInt64(0))
	}
	s := zed.DecodeString(val.Bytes())
	return ctx.CopyValue(zed.NewInt64(int64(utf8.RuneCountInString(s))))
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#lower
type ToLower struct {
	zctx *zed.Context
}

func (t *ToLower) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	val := args[0]
	if !val.IsString() {
		return newErrorf(t.zctx, ctx, "lower: string arg required")
	}
	if val.IsNull() {
		return zed.NullString
	}
	s := zed.DecodeString(val.Bytes())
	return newString(ctx, strings.ToLower(s))
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#upper
type ToUpper struct {
	zctx *zed.Context
}

func (t *ToUpper) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	val := args[0]
	if !val.IsString() {
		return newErrorf(t.zctx, ctx, "upper: string arg required")
	}
	if val.IsNull() {
		return zed.NullString
	}
	s := zed.DecodeString(val.Bytes())
	return newString(ctx, strings.ToUpper(s))
}

type Trim struct {
	zctx *zed.Context
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#trim
func (t *Trim) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	val := args[0]
	if !val.IsString() {
		return newErrorf(t.zctx, ctx, "trim: string arg required")
	}
	if val.IsNull() {
		return zed.NullString
	}
	s := zed.DecodeString(val.Bytes())
	return newString(ctx, strings.TrimSpace(s))
}

// // https://github.com/brimdata/zed/blob/main/docs/language/functions.md#split
type Split struct {
	zctx *zed.Context
	typ  zed.Type
}

func newSplit(zctx *zed.Context) *Split {
	return &Split{
		zctx: zctx,
		typ:  zctx.LookupTypeArray(zed.TypeString),
	}
}

func (s *Split) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	sVal := args[0]
	sepVal := args[1]
	if !sVal.IsString() || !sepVal.IsString() {
		return newErrorf(s.zctx, ctx, "split: string args required")
	}
	if sVal.IsNull() || sepVal.IsNull() {
		return ctx.NewValue(s.typ, nil)
	}
	str := zed.DecodeString(sVal.Bytes())
	sep := zed.DecodeString(sepVal.Bytes())
	splits := strings.Split(str, sep)
	var b zcode.Bytes
	for _, substr := range splits {
		b = zcode.Append(b, zed.EncodeString(substr))
	}
	return ctx.NewValue(s.typ, b)
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#join
type Join struct {
	zctx    *zed.Context
	builder strings.Builder
}

func (j *Join) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	splitsVal := args[0]
	typ, ok := zed.TypeUnder(splitsVal.Type).(*zed.TypeArray)
	if !ok {
		return newErrorf(j.zctx, ctx, "join: array of string args required")
	}
	if typ.Type.ID() != zed.IDString {
		return newErrorf(j.zctx, ctx, "join: array of string args required")
	}
	var separator string
	if len(args) == 2 {
		sepVal := args[1]
		if !sepVal.IsString() {
			return newErrorf(j.zctx, ctx, "join: separator must be string")
		}
		separator = zed.DecodeString(sepVal.Bytes())
	}
	b := j.builder
	b.Reset()
	it := splitsVal.Bytes().Iter()
	var sep string
	for !it.Done() {
		b.WriteString(sep)
		b.WriteString(zed.DecodeString(it.Next()))
		sep = separator
	}
	return newString(ctx, b.String())
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#levenshtein
type Levenshtein struct {
	zctx *zed.Context
}

func (l *Levenshtein) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	a, b := &args[0], &args[1]
	if !a.IsString() {
		return l.zctx.WrapError("levenshtein: string args required", a)
	}
	if !b.IsString() {
		return l.zctx.WrapError("levenshtein: string args required", b)
	}
	as, bs := zed.DecodeString(a.Bytes()), zed.DecodeString(b.Bytes())
	return ctx.CopyValue(zed.NewInt64(int64(levenshtein.ComputeDistance(as, bs))))
}

package function

import (
	"strings"
	"unicode/utf8"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#replace
type Replace struct{}

func (r *Replace) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	zvs := args[0]
	zvold := args[1]
	zvnew := args[2]
	if !zvs.IsStringy() || !zvold.IsStringy() || !zvnew.IsStringy() {
		return newErrorf(ctx, "replace: string arg required")
	}
	if zvs.Bytes == nil {
		return zed.Null
	}
	if zvold.Bytes == nil || zvnew.Bytes == nil {
		return newErrorf(ctx, "replace: an input arg is null")
	}
	s, err := zed.DecodeString(zvs.Bytes)
	if err != nil {
		panic(err)
	}
	old, err := zed.DecodeString(zvold.Bytes)
	if err != nil {
		panic(err)
	}
	new, err := zed.DecodeString(zvnew.Bytes)
	if err != nil {
		panic(err)
	}
	return newString(ctx, strings.ReplaceAll(s, old, new))
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#run_len
type RuneLen struct{}

func (r *RuneLen) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	zv := args[0]
	if !zv.IsStringy() {
		return newErrorf(ctx, "rune_len: string arg required")
	}
	if zv.Bytes == nil {
		return newInt64(ctx, 0)
	}
	in, err := zed.DecodeString(zv.Bytes)
	if err != nil {
		panic(err)
	}
	return newInt64(ctx, int64(utf8.RuneCountInString(in)))
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#to_lower
type ToLower struct{}

func (t *ToLower) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	zv := args[0]
	if !zv.IsStringy() {
		return newErrorf(ctx, "to_lower: string arg required")
	}
	if zv.IsNull() {
		return zed.NullString
	}
	s, err := zed.DecodeString(zv.Bytes)
	if err != nil {
		panic(err)
	}
	return newString(ctx, strings.ToLower(s))
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#to_upper
type ToUpper struct{}

func (t *ToUpper) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	zv := args[0]
	if !zv.IsStringy() {
		return newErrorf(ctx, "to_upper: string arg required")
	}
	if zv.IsNull() {
		return zed.NullString
	}
	s, err := zed.DecodeString(zv.Bytes)
	if err != nil {
		panic(err)
	}
	return newString(ctx, strings.ToUpper(s))
}

type Trim struct{}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#trim
func (t *Trim) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	zv := args[0]
	if !zv.IsStringy() {
		return newErrorf(ctx, "trim: string arg required")
	}
	if zv.IsNull() {
		return zed.NullString
	}
	s, err := zed.DecodeString(zv.Bytes)
	if err != nil {
		panic(err)
	}
	return newString(ctx, strings.TrimSpace(s))
}

// // https://github.com/brimdata/zed/blob/main/docs/language/functions.md#split
type Split struct {
	zctx *zed.Context
	typ  zed.Type
}

func newSplit(zctx *zed.Context) *Split {
	return &Split{
		typ: zctx.LookupTypeArray(zed.TypeString),
	}
}

func (s *Split) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	zs := args[0]
	zsep := args[1]
	if !zs.IsStringy() || !zsep.IsStringy() {
		return newErrorf(ctx, "split: string args required")
	}
	if zs.IsNull() || zsep.IsNull() {
		return ctx.NewValue(s.typ, nil)
	}
	str, err := zed.DecodeString(zs.Bytes)
	if err != nil {
		panic(err)
	}
	sep, err := zed.DecodeString(zsep.Bytes)
	if err != nil {
		panic(err)
	}
	splits := strings.Split(str, sep)
	var b zcode.Bytes
	for _, substr := range splits {
		b = zcode.AppendPrimitive(b, zed.EncodeString(substr))
	}
	return ctx.NewValue(s.typ, b)
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#join
type Join struct {
	builder strings.Builder
}

func (j *Join) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	zsplits := args[0]
	typ, ok := zed.AliasOf(zsplits.Type).(*zed.TypeArray)
	if !ok {
		return newErrorf(ctx, "join: array of string args required")
	}
	if !zed.IsStringy(typ.Type.ID()) {
		return newErrorf(ctx, "join: array of string args required")
	}
	var separator string
	if len(args) == 2 {
		zsep := args[1]
		if !zsep.IsStringy() {
			return newErrorf(ctx, "join: separator must be string")
		}
		var err error
		separator, err = zed.DecodeString(zsep.Bytes)
		if err != nil {
			panic(err)
		}
	}
	b := j.builder
	b.Reset()
	it := zsplits.Bytes.Iter()
	var sep string
	for !it.Done() {
		bytes, _, err := it.Next()
		if err != nil {
			panic(err)
		}
		s, err := zed.DecodeString(bytes)
		if err != nil {
			panic(err)
		}
		b.WriteString(sep)
		b.WriteString(s)
		sep = separator
	}
	return newString(ctx, b.String())
}

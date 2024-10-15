package function

import (
	"strings"
	"unicode/utf8"

	"github.com/agnivade/levenshtein"
	"github.com/brimdata/super"
	"github.com/brimdata/super/zcode"
)

// https://github.com/brimdata/super/blob/main/docs/language/functions.md#replace
type Replace struct {
	zctx *zed.Context
}

func (r *Replace) Call(_ zed.Allocator, args []zed.Value) zed.Value {
	args = underAll(args)
	sVal := args[0]
	oldVal := args[1]
	newVal := args[2]
	for i := range args {
		if !args[i].IsString() {
			return r.zctx.WrapError("replace: string arg required", args[i])
		}
	}
	if sVal.IsNull() {
		return zed.Null
	}
	if oldVal.IsNull() || newVal.IsNull() {
		return r.zctx.NewErrorf("replace: an input arg is null")
	}
	s := zed.DecodeString(sVal.Bytes())
	old := zed.DecodeString(oldVal.Bytes())
	new := zed.DecodeString(newVal.Bytes())
	return zed.NewString(strings.ReplaceAll(s, old, new))
}

// https://github.com/brimdata/super/blob/main/docs/language/functions.md#run_len
type RuneLen struct {
	zctx *zed.Context
}

func (r *RuneLen) Call(_ zed.Allocator, args []zed.Value) zed.Value {
	val := args[0].Under()
	if !val.IsString() {
		return r.zctx.WrapError("rune_len: string arg required", val)
	}
	if val.IsNull() {
		return zed.NewInt64(0)
	}
	s := zed.DecodeString(val.Bytes())
	return zed.NewInt64(int64(utf8.RuneCountInString(s)))
}

// https://github.com/brimdata/super/blob/main/docs/language/functions.md#lower
type ToLower struct {
	zctx *zed.Context
}

func (t *ToLower) Call(_ zed.Allocator, args []zed.Value) zed.Value {
	val := args[0].Under()
	if !val.IsString() {
		return t.zctx.WrapError("lower: string arg required", val)
	}
	if val.IsNull() {
		return zed.NullString
	}
	s := zed.DecodeString(val.Bytes())
	return zed.NewString(strings.ToLower(s))
}

// https://github.com/brimdata/super/blob/main/docs/language/functions.md#upper
type ToUpper struct {
	zctx *zed.Context
}

func (t *ToUpper) Call(_ zed.Allocator, args []zed.Value) zed.Value {
	val := args[0].Under()
	if !val.IsString() {
		return t.zctx.WrapError("upper: string arg required", val)
	}
	if val.IsNull() {
		return zed.NullString
	}
	s := zed.DecodeString(val.Bytes())
	return zed.NewString(strings.ToUpper(s))
}

type Trim struct {
	zctx *zed.Context
}

// https://github.com/brimdata/super/blob/main/docs/language/functions.md#trim
func (t *Trim) Call(_ zed.Allocator, args []zed.Value) zed.Value {
	val := args[0].Under()
	if !val.IsString() {
		return t.zctx.WrapError("trim: string arg required", val)
	}
	if val.IsNull() {
		return zed.NullString
	}
	s := zed.DecodeString(val.Bytes())
	return zed.NewString(strings.TrimSpace(s))
}

// // https://github.com/brimdata/super/blob/main/docs/language/functions.md#split
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

func (s *Split) Call(_ zed.Allocator, args []zed.Value) zed.Value {
	args = underAll(args)
	for i := range args {
		if !args[i].IsString() {
			return s.zctx.WrapError("split: string arg required", args[i])
		}
	}
	sVal, sepVal := args[0], args[1]
	if sVal.IsNull() || sepVal.IsNull() {
		return zed.NewValue(s.typ, nil)
	}
	str := zed.DecodeString(sVal.Bytes())
	sep := zed.DecodeString(sepVal.Bytes())
	splits := strings.Split(str, sep)
	var b zcode.Bytes
	for _, substr := range splits {
		b = zcode.Append(b, zed.EncodeString(substr))
	}
	return zed.NewValue(s.typ, b)
}

// https://github.com/brimdata/super/blob/main/docs/language/functions.md#join
type Join struct {
	zctx    *zed.Context
	builder strings.Builder
}

func (j *Join) Call(_ zed.Allocator, args []zed.Value) zed.Value {
	args = underAll(args)
	splitsVal := args[0]
	typ, ok := zed.TypeUnder(splitsVal.Type()).(*zed.TypeArray)
	if !ok || typ.Type.ID() != zed.IDString {
		return j.zctx.WrapError("join: array of string arg required", splitsVal)
	}
	var separator string
	if len(args) == 2 {
		sepVal := args[1]
		if !sepVal.IsString() {
			return j.zctx.WrapError("join: separator must be string", sepVal)
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
	return zed.NewString(b.String())
}

// https://github.com/brimdata/super/blob/main/docs/language/functions.md#levenshtein
type Levenshtein struct {
	zctx *zed.Context
}

func (l *Levenshtein) Call(_ zed.Allocator, args []zed.Value) zed.Value {
	args = underAll(args)
	a, b := args[0], args[1]
	if !a.IsString() {
		return l.zctx.WrapError("levenshtein: string args required", a)
	}
	if !b.IsString() {
		return l.zctx.WrapError("levenshtein: string args required", b)
	}
	as, bs := zed.DecodeString(a.Bytes()), zed.DecodeString(b.Bytes())
	return zed.NewInt64(int64(levenshtein.ComputeDistance(as, bs)))
}

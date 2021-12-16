package function

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr/result"
	"github.com/brimdata/zed/zcode"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#replace
type Replace struct {
	stash result.Value
}

func (r *Replace) Call(args []zed.Value) *zed.Value {
	zvs := args[0]
	zvold := args[1]
	zvnew := args[2]
	if !zvs.IsStringy() || !zvold.IsStringy() || !zvnew.IsStringy() {
		return r.stash.Error(errors.New("replace: string arg required"))
	}
	if zvs.Bytes == nil {
		return zed.Null
	}
	if zvold.Bytes == nil || zvnew.Bytes == nil {
		return r.stash.Error(errors.New("replace: an input arg is null"))
	}
	s, err := zed.DecodeString(zvs.Bytes)
	if err != nil {
		panic(fmt.Errorf("replace: corrupt Zed bytes: %w", err))
	}
	old, err := zed.DecodeString(zvold.Bytes)
	if err != nil {
		panic(fmt.Errorf("replace: corrupt Zed bytes: %w", err))
	}
	new, err := zed.DecodeString(zvnew.Bytes)
	if err != nil {
		panic(fmt.Errorf("replace: corrupt Zed bytes: %w", err))
	}
	result := strings.ReplaceAll(s, old, new)
	return r.stash.String(result)
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#run_len
type RuneLen struct {
	stash result.Value
}

func (r *RuneLen) Call(args []zed.Value) *zed.Value {
	zv := args[0]
	if !zv.IsStringy() {
		return r.stash.Error(errors.New("rune_len: string arg required"))
	}
	if zv.Bytes == nil {
		return r.stash.Int64(0)
	}
	in, err := zed.DecodeString(zv.Bytes)
	if err != nil {
		panic(fmt.Errorf("rune_len: corrupt Zed bytes: %w", err))
	}
	return r.stash.Int64(int64(utf8.RuneCountInString(in)))
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#to_lower
type ToLower struct {
	stash result.Value
}

func (t *ToLower) Call(args []zed.Value) *zed.Value {
	zv := args[0]
	if !zv.IsStringy() {
		return t.stash.Error(errors.New("to_lower: string arg required"))
	}
	if zv.IsNull() {
		return zed.NullString
	}
	s, err := zed.DecodeString(zv.Bytes)
	if err != nil {
		panic(fmt.Errorf("to_lower: corrupt Zed bytes: %w", err))
	}
	// XXX GC
	return t.stash.String(strings.ToLower(s))
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#to_upper
type ToUpper struct {
	stash result.Value
}

func (t *ToUpper) Call(args []zed.Value) *zed.Value {
	zv := args[0]
	if !zv.IsStringy() {
		return t.stash.Error(errors.New("to_upper: string arg required"))
	}
	if zv.IsNull() {
		return zed.NullString
	}
	s, err := zed.DecodeString(zv.Bytes)
	if err != nil {
		panic(fmt.Errorf("to_upper: corrupt Zed bytes: %w", err))
	}
	// XXX GC
	return t.stash.String(strings.ToUpper(s))
}

type Trim struct {
	stash result.Value
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#trim
func (t *Trim) Call(args []zed.Value) *zed.Value {
	zv := args[0]
	if !zv.IsStringy() {
		return t.stash.Error(errors.New("trim: string arg required"))
	}
	if zv.IsNull() {
		return zed.NullString
	}
	s, err := zed.DecodeString(zv.Bytes)
	if err != nil {
		panic(fmt.Errorf("trim: corrupt Zed bytes: %w", err))
	}
	// XXX GC
	return t.stash.String(strings.TrimSpace(s))
}

// // https://github.com/brimdata/zed/blob/main/docs/language/functions.md#split
type Split struct {
	zctx  *zed.Context
	typ   zed.Type
	stash result.Value
}

func newSplit(zctx *zed.Context) *Split {
	return &Split{
		typ: zctx.LookupTypeArray(zed.TypeString),
	}
}

func (s *Split) Call(args []zed.Value) *zed.Value {
	zs := args[0]
	zsep := args[1]
	if !zs.IsStringy() || !zsep.IsStringy() {
		return s.stash.Error(errors.New("split: string args required"))
	}
	if zs.IsNull() || zsep.IsNull() {
		return s.stash.CopyVal(zed.Value{Type: s.typ})
	}
	str, err := zed.DecodeString(zs.Bytes)
	if err != nil {
		panic(fmt.Errorf("split: corrupt Zed bytes: %w", err))
	}
	sep, err := zed.DecodeString(zsep.Bytes)
	if err != nil {
		panic(fmt.Errorf("split: corrupt Zed bytes: %w", err))
	}
	splits := strings.Split(str, sep)
	b := s.stash.Bytes[:0]
	for _, substr := range splits {
		b = zcode.AppendPrimitive(b, zed.EncodeString(substr))
	}
	s.stash.Bytes = b
	s.stash.Type = s.typ
	return (*zed.Value)(&s.stash)
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#join
type Join struct {
	stash   result.Value
	builder strings.Builder
}

func (j *Join) Call(args []zed.Value) *zed.Value {
	zsplits := args[0]
	typ, ok := zed.AliasOf(zsplits.Type).(*zed.TypeArray)
	if !ok {
		return j.stash.Error(errors.New("join: array of string args required"))
	}
	if !zed.IsStringy(typ.Type.ID()) {
		return j.stash.Error(errors.New("join: array of string args required"))
	}
	var separator string
	if len(args) == 2 {
		zsep := args[1]
		if !zsep.IsStringy() {
			return j.stash.Error(errors.New("join: separator must be string"))
		}
		var err error
		separator, err = zed.DecodeString(zsep.Bytes)
		if err != nil {
			panic(fmt.Errorf("join: corrupt Zed bytes: %w", err))
		}
	}
	b := j.builder
	b.Reset()
	it := zsplits.Bytes.Iter()
	var sep string
	for !it.Done() {
		bytes, _, err := it.Next()
		if err != nil {
			panic(fmt.Errorf("join: corrupt Zed bytes: %w", err))
		}
		s, err := zed.DecodeString(bytes)
		if err != nil {
			panic(fmt.Errorf("join: corrupt Zed bytes: %w", err))
		}
		b.WriteString(sep)
		b.WriteString(s)
		sep = separator
	}
	return j.stash.String(b.String())
}

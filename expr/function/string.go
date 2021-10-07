package function

import (
	"strings"
	"unicode/utf8"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr/result"
	"github.com/brimdata/zed/zcode"
)

type replace struct{}

func (*replace) Call(args []zed.Value) (zed.Value, error) {
	zvs := args[0]
	zvold := args[1]
	zvnew := args[2]
	if !zvs.IsStringy() || !zvold.IsStringy() || !zvnew.IsStringy() {
		return badarg("replace")
	}
	if zvs.Bytes == nil {
		return zvs, nil
	}
	if zvold.Bytes == nil || zvnew.Bytes == nil {
		return badarg("replace")
	}
	s, err := zed.DecodeString(zvs.Bytes)
	if err != nil {
		return zed.Value{}, err
	}
	old, err := zed.DecodeString(zvold.Bytes)
	if err != nil {
		return zed.Value{}, err
	}
	new, err := zed.DecodeString(zvnew.Bytes)
	if err != nil {
		return zed.Value{}, err
	}
	result := strings.ReplaceAll(s, old, new)
	return zed.Value{zed.TypeString, zed.EncodeString(result)}, nil
}

type runeLen struct {
	result.Buffer
}

func (s *runeLen) Call(args []zed.Value) (zed.Value, error) {
	zv := args[0]
	if !zv.IsStringy() {
		return badarg("rune_len")
	}
	if zv.Bytes == nil {
		return zed.Value{zed.TypeInt64, s.Int(0)}, nil
	}
	in, err := zed.DecodeString(zv.Bytes)
	if err != nil {
		return zed.Value{}, err
	}
	v := utf8.RuneCountInString(in)
	return zed.Value{zed.TypeInt64, s.Int(int64(v))}, nil
}

type toLower struct{}

func (*toLower) Call(args []zed.Value) (zed.Value, error) {
	zv := args[0]
	if !zv.IsStringy() {
		return badarg("to_lower")
	}
	if zv.Bytes == nil {
		return zv, nil
	}
	s, err := zed.DecodeString(zv.Bytes)
	if err != nil {
		return zed.Value{}, err
	}
	// XXX GC
	s = strings.ToLower(s)
	return zed.Value{zed.TypeString, zed.EncodeString(s)}, nil
}

type toUpper struct{}

func (*toUpper) Call(args []zed.Value) (zed.Value, error) {
	zv := args[0]
	if !zv.IsStringy() {
		return badarg("to_upper")
	}
	if zv.Bytes == nil {
		return zv, nil
	}
	s, err := zed.DecodeString(zv.Bytes)
	if err != nil {
		return zed.Value{}, err
	}
	// XXX GC
	s = strings.ToUpper(s)
	return zed.Value{zed.TypeString, zed.EncodeString(s)}, nil
}

type trim struct{}

func (*trim) Call(args []zed.Value) (zed.Value, error) {
	zv := args[0]
	if !zv.IsStringy() {
		return badarg("trim")
	}
	if zv.Bytes == nil {
		return zv, nil
	}
	// XXX GC
	s := strings.TrimSpace(string(zv.Bytes))
	return zed.Value{zed.TypeString, zed.EncodeString(s)}, nil
}

type split struct {
	zctx  *zed.Context
	typ   zed.Type
	bytes zcode.Bytes
}

func newSplit(zctx *zed.Context) *split {
	return &split{
		typ: zctx.LookupTypeArray(zed.TypeString),
	}
}

func (s *split) Call(args []zed.Value) (zed.Value, error) {
	zs := args[0]
	zsep := args[1]
	if !zs.IsStringy() || !zsep.IsStringy() {
		return badarg("split")
	}
	if zs.Bytes == nil || zsep.Bytes == nil {
		return zed.Value{Type: s.typ}, nil
	}
	str, err := zed.DecodeString(zs.Bytes)
	if err != nil {
		return zed.Value{}, err
	}
	sep, err := zed.DecodeString(zsep.Bytes)
	if err != nil {
		return zed.Value{}, err
	}
	splits := strings.Split(str, sep)
	b := s.bytes[:0]
	for _, substr := range splits {
		b = zcode.AppendPrimitive(b, zed.EncodeString(substr))
	}
	s.bytes = b
	return zed.Value{s.typ, b}, nil
}

type join struct {
	bytes   zcode.Bytes
	builder strings.Builder
}

func (j *join) Call(args []zed.Value) (zed.Value, error) {
	zsplits := args[0]
	typ, ok := zed.AliasOf(zsplits.Type).(*zed.TypeArray)
	if !ok {
		return zed.NewErrorf("argument to join() is not an array"), nil
	}
	if !zed.IsStringy(typ.Type.ID()) {
		return zed.NewErrorf("argument to join() is not a string array"), nil
	}
	var separator string
	if len(args) == 2 {
		zsep := args[1]
		if !zsep.IsStringy() {
			return zed.NewErrorf("separator argument to join() is not a string"), nil
		}
		var err error
		separator, err = zed.DecodeString(zsep.Bytes)
		if err != nil {
			return zed.Value{}, err
		}
	}
	b := j.builder
	b.Reset()
	it := zsplits.Bytes.Iter()
	var sep string
	for !it.Done() {
		bytes, _, err := it.Next()
		if err != nil {
			return zed.Value{}, err
		}
		s, err := zed.DecodeString(bytes)
		if err != nil {
			return zed.Value{}, err
		}
		b.WriteString(sep)
		b.WriteString(s)
		sep = separator
	}
	return zed.Value{zed.TypeString, zed.EncodeString(b.String())}, nil
}

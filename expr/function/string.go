package function

import (
	"net"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr/result"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

// XXX these string format functions should be handlded by :string cast

type stringFormatFloat struct{}

func (s *stringFormatFloat) Call(args []zed.Value) (zed.Value, error) {
	zv := args[0]
	if zv.Type.ID() != zed.IDFloat64 {
		return badarg("string.floatToString")
	}
	if zv.Bytes == nil {
		return zed.Value{zed.TypeString, nil}, nil
	}
	f, _ := zed.DecodeFloat64(zv.Bytes)
	// XXX GC
	v := strconv.FormatFloat(f, 'g', -1, 64)
	return zed.Value{zed.TypeString, zed.EncodeString(v)}, nil
}

type stringFormatInt struct{}

func (s *stringFormatInt) Call(args []zed.Value) (zed.Value, error) {
	zv := args[0]
	id := zv.Type.ID()
	var out string
	if !zed.IsInteger(id) {
		return badarg("string.intToString")
	}
	if zv.Bytes == nil {
		return zed.Value{zed.TypeString, nil}, nil
	}
	if zed.IsSigned(id) {
		v, _ := zed.DecodeInt(zv.Bytes)
		// XXX GC
		out = strconv.FormatInt(v, 10)
	} else {
		v, _ := zed.DecodeUint(zv.Bytes)
		// XXX GC
		out = strconv.FormatUint(v, 10)
	}
	return zed.Value{zed.TypeString, zed.EncodeString(out)}, nil
}

type stringFormatIp struct{}

func (s *stringFormatIp) Call(args []zed.Value) (zed.Value, error) {
	zv := args[0]
	if zv.Type.ID() != zed.IDIP {
		return badarg("string.ipToString")
	}
	if zv.Bytes == nil {
		return zed.Value{zed.TypeString, nil}, nil
	}
	// XXX GC
	ip, _ := zed.DecodeIP(zv.Bytes)
	return zed.Value{zed.TypeString, zed.EncodeString(ip.String())}, nil
}

type stringParseInt struct {
	result.Buffer
}

func (s *stringParseInt) Call(args []zed.Value) (zed.Value, error) {
	zv := args[0]
	if !zv.IsStringy() {
		return badarg("String.parseInt")
	}
	if zv.Bytes == nil {
		return zed.Value{zed.TypeInt64, nil}, nil
	}
	v, e := zed.DecodeString(zv.Bytes)
	if e != nil {
		return zed.Value{}, e
	}
	i, perr := strconv.ParseInt(v, 10, 64)
	if perr != nil {
		// Get rid of the strconv wrapping gunk to get the
		// actual error message
		e := perr.(*strconv.NumError)
		return zverr("String.parseInt", e.Err)
	}
	return zed.Value{zed.TypeInt64, s.Int(i)}, nil
}

type stringParseFloat struct {
	result.Buffer
}

func (s *stringParseFloat) Call(args []zed.Value) (zed.Value, error) {
	zv := args[0]
	if !zv.IsStringy() {
		return badarg("String.parseFloat")
	}
	if zv.Bytes == nil {
		return zed.Value{zed.TypeFloat64, nil}, nil
	}
	v, perr := zed.DecodeString(zv.Bytes)
	if perr != nil {
		return zed.Value{}, perr
	}
	f, perr := strconv.ParseFloat(v, 64)
	if perr != nil {
		// Get rid of the strconv wrapping gunk to get the
		// actual error message
		e := perr.(*strconv.NumError)
		return zverr("String.parseFloat", e.Err)
	}
	return zed.Value{zed.TypeFloat64, s.Float64(f)}, nil
}

type stringParseIp struct{}

func (s *stringParseIp) Call(args []zed.Value) (zed.Value, error) {
	zv := args[0]
	if !zv.IsStringy() {
		return badarg("String.parseIp")
	}
	if zv.Bytes == nil {
		return zed.Value{zed.TypeIP, nil}, nil
	}
	v, err := zed.DecodeString(zv.Bytes)
	if err != nil {
		return zed.Value{}, err
	}
	// XXX GC
	a := net.ParseIP(v)
	if a == nil {
		return badarg("String.parseIp")
	}
	// XXX GC
	return zed.Value{zed.TypeIP, zed.EncodeIP(a)}, nil
}

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
	zctx  *zson.Context
	typ   zed.Type
	bytes zcode.Bytes
}

func newSplit(zctx *zson.Context) *split {
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

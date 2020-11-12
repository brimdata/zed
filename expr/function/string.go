package function

import (
	"net"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/brimsec/zq/expr/result"
	"github.com/brimsec/zq/zng"
)

type bytelen struct {
	result.Buffer
}

// XXX we should just have a len function that applies to different types
// and a way to get unicode char len, charlen()?
func (b *bytelen) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	if !zv.IsStringy() {
		return badarg("Strings.byteLen")
	}
	v := len(string(zv.Bytes))
	return zng.Value{zng.TypeInt64, b.Int(int64(v))}, nil
}

// XXX these string format functions should be handlded by :string cast

type stringFormatFloat struct{}

func (s *stringFormatFloat) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	if zv.Type.ID() != zng.IdFloat64 {
		return badarg("string.floatToString")
	}
	f, _ := zng.DecodeFloat64(zv.Bytes)
	// XXX GC
	v := strconv.FormatFloat(f, 'g', -1, 64)
	return zng.Value{zng.TypeString, zng.EncodeString(v)}, nil
}

type stringFormatInt struct{}

func (s *stringFormatInt) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	id := zv.Type.ID()
	var out string
	if !zng.IsInteger(id) {
		return badarg("string.intToString")
	}
	if zng.IsSigned(id) {
		v, _ := zng.DecodeInt(zv.Bytes)
		// XXX GC
		out = strconv.FormatInt(v, 10)
	} else {
		v, _ := zng.DecodeUint(zv.Bytes)
		// XXX GC
		out = strconv.FormatUint(v, 10)
	}
	return zng.Value{zng.TypeString, zng.EncodeString(out)}, nil
}

type stringFormatIp struct{}

func (s *stringFormatIp) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	if zv.Type.ID() != zng.IdIP {
		return badarg("string.ipToString")
	}
	// XXX GC
	ip, _ := zng.DecodeIP(zv.Bytes)
	return zng.Value{zng.TypeString, zng.EncodeString(ip.String())}, nil
}

type stringParseInt struct {
	result.Buffer
}

func (s *stringParseInt) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	if !zv.IsStringy() {
		return badarg("String.parseInt")
	}
	v, e := zng.DecodeString(zv.Bytes)
	if e != nil {
		return zng.Value{}, e
	}
	i, perr := strconv.ParseInt(v, 10, 64)
	if perr != nil {
		// Get rid of the strconv wrapping gunk to get the
		// actual error message
		e := perr.(*strconv.NumError)
		return zverr("String.parseInt", e.Err)
	}
	return zng.Value{zng.TypeInt64, s.Int(i)}, nil
}

type stringParseFloat struct {
	result.Buffer
}

func (s *stringParseFloat) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	if !zv.IsStringy() {
		return badarg("String.parseFloat")
	}
	v, perr := zng.DecodeString(zv.Bytes)
	if perr != nil {
		return zng.Value{}, perr
	}
	f, perr := strconv.ParseFloat(v, 64)
	if perr != nil {
		// Get rid of the strconv wrapping gunk to get the
		// actual error message
		e := perr.(*strconv.NumError)
		return zverr("String.parseFloat", e.Err)
	}
	return zng.Value{zng.TypeFloat64, s.Float64(f)}, nil
}

type stringParseIp struct{}

func (s *stringParseIp) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	if !zv.IsStringy() {
		return badarg("String.parseIp")
	}
	v, err := zng.DecodeString(zv.Bytes)
	if err != nil {
		return zng.Value{}, err
	}
	// XXX GC
	a := net.ParseIP(v)
	if a == nil {
		return badarg("String.parseIp")
	}
	// XXX GC
	return zng.Value{zng.TypeIP, zng.EncodeIP(a)}, nil
}

type replace struct{}

func (*replace) Call(args []zng.Value) (zng.Value, error) {
	zvs := args[0]
	zvold := args[1]
	zvnew := args[2]
	if !zvs.IsStringy() || !zvold.IsStringy() || !zvnew.IsStringy() {
		return badarg("replace")
	}
	s, err := zng.DecodeString(zvs.Bytes)
	if err != nil {
		return zng.Value{}, err
	}
	old, err := zng.DecodeString(zvold.Bytes)
	if err != nil {
		return zng.Value{}, err
	}
	new, err := zng.DecodeString(zvnew.Bytes)
	if err != nil {
		return zng.Value{}, err
	}
	result := strings.ReplaceAll(s, old, new)
	return zng.Value{zng.TypeString, zng.EncodeString(result)}, nil
}

type runeLen struct {
	result.Buffer
}

func (s *runeLen) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	if !zv.IsStringy() {
		return badarg("Strings.byteLen")
	}
	in, err := zng.DecodeString(zv.Bytes)
	if err != nil {
		return zng.Value{}, err
	}
	v := utf8.RuneCountInString(in)
	return zng.Value{zng.TypeInt64, s.Int(int64(v))}, nil
}

type toLower struct{}

func (*toLower) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	if !zv.IsStringy() {
		return badarg("to_lower")
	}
	s, err := zng.DecodeString(zv.Bytes)
	if err != nil {
		return zng.Value{}, err
	}
	// XXX GC
	s = strings.ToLower(s)
	return zng.Value{zng.TypeString, zng.EncodeString(s)}, nil
}

type toUpper struct{}

func (*toUpper) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	if !zv.IsStringy() {
		return badarg("to_upper")
	}
	s, err := zng.DecodeString(zv.Bytes)
	if err != nil {
		return zng.Value{}, err
	}
	// XXX GC
	s = strings.ToUpper(s)
	return zng.Value{zng.TypeString, zng.EncodeString(s)}, nil
}

type trim struct{}

func (*trim) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	if !zv.IsStringy() {
		return badarg("trim")
	}
	// XXX GC
	s := strings.TrimSpace(string(zv.Bytes))
	return zng.Value{zng.TypeString, zng.EncodeString(s)}, nil
}

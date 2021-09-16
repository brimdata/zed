package function

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed/anymath"
	"github.com/brimdata/zed/expr/result"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

var (
	ErrBadArgument    = errors.New("bad argument")
	ErrNoSuchFunction = errors.New("no such function")
	ErrTooFewArgs     = errors.New("too few arguments")
	ErrTooManyArgs    = errors.New("too many arguments")
)

type Interface interface {
	Call([]zng.Value) (zng.Value, error)
}

func New(zctx *zson.Context, name string, narg int) (Interface, bool, error) {
	argmin := 1
	argmax := 1
	var root bool
	var f Interface
	switch name {
	default:
		return nil, false, ErrNoSuchFunction
	case "len":
		f = &lenFn{}
	case "abs":
		f = &abs{}
	case "ceil":
		f = &ceil{}
	case "floor":
		f = &floor{}
	case "join":
		argmax = 2
		f = &join{}
	case "ksuid":
		f = &ksuidToString{}
	case "log":
		f = &log{}
	case "max":
		argmax = -1
		f = &reducer{fn: anymath.Max}
	case "min":
		argmax = -1
		f = &reducer{fn: anymath.Min}
	case "mod":
		argmin = 2
		argmax = 2
		f = &mod{}
	case "round":
		f = &round{}
	case "pow":
		argmin = 2
		argmax = 2
		f = &pow{}
	case "sqrt":
		f = &sqrt{}
	case "replace":
		argmin = 3
		argmax = 3
		f = &replace{}
	case "rune_len":
		f = &runeLen{}
	case "to_lower":
		f = &toLower{}
	case "to_upper":
		f = &toUpper{}
	case "trim":
		f = &trim{}
	case "iso":
		f = &iso{}
	case "split":
		argmin = 2
		argmax = 2
		f = newSplit(zctx)
	case "trunc":
		argmin = 2
		argmax = 2
		f = &trunc{}
	case "typeof":
		f = &typeOf{zctx}
	case "typeunder":
		f = &typeUnder{zctx}
	case "nameof":
		f = &nameOf{}
	case "fields":
		typ := zctx.LookupTypeArray(zng.TypeString)
		f = &fields{zctx: zctx, typ: typ}
	case "is":
		argmin = 1
		argmax = 2
		root = true
		f = &is{zctx: zctx}
	case "iserr":
		f = &isErr{}
	case "to_base64":
		f = &toBase64{}
	case "from_base64":
		f = &fromBase64{}
	case "to_hex":
		f = &toHex{}
	case "from_hex":
		f = &fromHex{}
	case "network_of":
		argmax = 2
		f = &networkOf{}
	case "parse_uri":
		f = &parseURI{marshaler: zson.NewZNGMarshalerWithContext(zctx)}
	case "parse_zson":
		f = &parseZSON{zctx: zctx}
	}
	if argmin != -1 && narg < argmin {
		return nil, false, ErrTooFewArgs
	}
	if argmax != -1 && narg > argmax {
		return nil, false, ErrTooManyArgs
	}
	return f, root, nil
}

// HasBoolResult returns true if the function name returns a Boolean value.
// XXX This is a hack so the semantic compiler can determine if a single call
// expr is a Filter or Put proc. At some point function declarations should have
// signatures so the return type can be introspected.
func HasBoolResult(name string) bool {
	switch name {
	case "iserr", "is", "has", "missing":
		return true
	}
	return false
}

func zverr(msg string, err error) (zng.Value, error) {
	return zng.Value{}, fmt.Errorf("%s: %w", msg, err)
}

func badarg(msg string) (zng.Value, error) {
	return zverr(msg, ErrBadArgument)
}

type lenFn struct {
	result.Buffer
}

func (l *lenFn) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	if zv.Bytes == nil {
		return zng.Value{zng.TypeInt64, nil}, nil
	}
	switch typ := zng.AliasOf(args[0].Type).(type) {
	case *zng.TypeRecord:
		return zng.Value{zng.TypeInt64, l.Int(int64(len(typ.Columns)))}, nil
	case *zng.TypeArray, *zng.TypeSet:
		len, err := zv.ContainerLength()
		if err != nil {
			return zng.Value{}, err
		}
		return zng.Value{zng.TypeInt64, l.Int(int64(len))}, nil
	case *zng.TypeOfString, *zng.TypeOfBstring, *zng.TypeOfIP, *zng.TypeOfNet:
		v := len(zv.Bytes)
		return zng.Value{zng.TypeInt64, l.Int(int64(v))}, nil
	default:
		return badarg("len")
	}
}

type typeOf struct {
	zctx *zson.Context
}

func (t *typeOf) Call(args []zng.Value) (zng.Value, error) {
	typ := args[0].Type
	return t.zctx.LookupTypeValue(typ), nil
}

type typeUnder struct {
	zctx *zson.Context
}

func (t *typeUnder) Call(args []zng.Value) (zng.Value, error) {
	typ := zng.AliasOf(args[0].Type)
	return t.zctx.LookupTypeValue(typ), nil
}

type nameOf struct{}

func (*nameOf) Call(args []zng.Value) (zng.Value, error) {
	typ := args[0].Type
	if alias, ok := typ.(*zng.TypeAlias); ok {
		// XXX GC
		return zng.Value{zng.TypeString, zng.EncodeString(alias.Name)}, nil
	}
	return zng.Missing, nil
}

type isErr struct{}

func (*isErr) Call(args []zng.Value) (zng.Value, error) {
	if args[0].IsError() {
		return zng.True, nil
	}
	return zng.False, nil
}

type is struct {
	zctx *zson.Context
}

func (i *is) Call(args []zng.Value) (zng.Value, error) {
	zvSubject := args[0]
	zvTypeVal := args[1]
	if len(args) == 3 {
		zvSubject = args[1]
		zvTypeVal = args[2]
	}
	var typ zng.Type
	var err error
	if zvTypeVal.IsStringy() {
		typ, err = zson.ParseType(i.zctx, string(zvTypeVal.Bytes))
	} else {
		typ, err = i.zctx.LookupByValue(zvTypeVal.Bytes)
	}
	if err == nil && typ == zvSubject.Type {
		return zng.True, nil
	}
	return zng.False, nil
}

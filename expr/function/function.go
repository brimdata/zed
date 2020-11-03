package function

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/anymath"
	"github.com/brimsec/zq/expr/result"
	"github.com/brimsec/zq/zng"
)

var (
	ErrTooFewArgs     = errors.New("too few arguments")
	ErrTooManyArgs    = errors.New("too many arguments")
	ErrBadArgument    = errors.New("bad argument")
	ErrNoSuchFunction = errors.New("no such function")
)

type Interface interface {
	Call([]zng.Value) (zng.Value, error)
}

func New(name string, narg int) (Interface, error) {
	argmin := 1
	argmax := 1
	var f Interface
	switch name {
	default:
		return nil, ErrNoSuchFunction
	case "len":
		f = &lenFn{}
	case "Math.abs", "abs":
		f = &abs{}
	case "Math.ceil", "ceil":
		f = &ceil{}
	case "Math.floor", "floor":
		f = &floor{}
	case "Math.log", "log":
		f = &log{}
	case "Math.max", "max":
		argmax = -1
		f = &reducer{fn: anymath.Max}
	case "Math.min", "min":
		argmax = -1
		f = &reducer{fn: anymath.Min}
	case "Math.mod", "mod":
		argmin = 2
		argmax = 2
		f = &mod{}
	case "Math.round", "round":
		f = &round{}
	case "Math.pow", "pow":
		argmin = 2
		argmax = 2
		f = &pow{}
	case "Math.sqrt", "sqrt":
		f = &sqrt{}
	case "String.byteLen":
		f = &bytelen{}
	case "String.formatFloat":
		// deprecated by <float-val>:string
		f = &stringFormatFloat{}
	case "String.formatInt":
		// deprecated by <int-val>:string
		f = &stringFormatInt{}
	case "String.formatIp":
		// deprecated by <ip-val>:string
		f = &stringFormatIp{}
	case "String.parseFloat":
		// deprecated by <string-val>:float
		f = &stringParseFloat{}
	case "String.parseInt":
		// deprecated by <string-val>:int<n>
		f = &stringParseInt{}
	case "String.parseIp":
		// deprecated by <string-val>:ip
		f = &stringParseIp{}
	case "String.replace", "replace":
		argmin = 3
		argmax = 3
		f = &replace{}
	case "String.runeLen", "rune_len":
		f = &rune_len{}
	case "String.toLower", "to_lower":
		f = &to_lower{}
	case "String.toUpper", "to_upper":
		f = &to_upper{}
	case "String.trim", "trim":
		f = &trim{}
	case "Time.fromISO", "iso":
		f = &iso{}
	case "Time.fromMilliseconds", "ms":
		f = &ms{}
	case "Time.fromMicroseconds", "us":
		f = &us{}
	case "Time.fromNanoseconds", "ns":
		f = &ns{}
	case "Time.trunc", "trunc":
		argmin = 2
		argmax = 2
		f = &trunc{}
	case "typeof":
		f = &typeOf{}
	case "iserr":
		f = &isErr{}
	case "toBase64", "to_base64":
		f = &to_base64{}
	case "fromBase64", "from_base64":
		f = &from_base64{}
	}
	if argmin != -1 && narg < argmin {
		return nil, ErrTooFewArgs
	}
	if argmax != -1 && narg > argmax {
		return nil, ErrTooManyArgs
	}
	return f, nil
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
	switch zng.AliasedType(args[0].Type).(type) {
	case *zng.TypeArray, *zng.TypeSet:
		v := args[0]
		len, err := v.ContainerLength()
		if err != nil {
			return zng.Value{}, err
		}
		return zng.Value{zng.TypeInt64, l.Int(int64(len))}, nil
	default:
		return badarg("len")
	}
}

type typeOf struct{}

func (t *typeOf) Call(args []zng.Value) (zng.Value, error) {
	return zng.Value{zng.TypeType, zng.EncodeType(args[0].Type.String())}, nil
}

type isErr struct{}

func (*isErr) Call(args []zng.Value) (zng.Value, error) {
	if args[0].IsError() {
		return zng.True, nil
	}
	return zng.False, nil
}

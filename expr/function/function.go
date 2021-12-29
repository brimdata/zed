package function

import (
	"errors"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/anymath"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zson"
)

var (
	ErrBadArgument    = errors.New("bad argument")
	ErrNoSuchFunction = errors.New("no such function")
	ErrTooFewArgs     = errors.New("too few arguments")
	ErrTooManyArgs    = errors.New("too many arguments")
)

type Interface interface {
	Call(zed.Allocator, []zed.Value) *zed.Value
}

func New(zctx *zed.Context, name string, narg int) (Interface, bool, error) {
	argmin := 1
	argmax := 1
	var wantThis bool
	var f Interface
	switch name {
	default:
		return nil, false, ErrNoSuchFunction
	case "len":
		f = &LenFn{}
	case "abs":
		f = &Abs{}
	case "ceil":
		f = &Ceil{}
	case "floor":
		f = &Floor{}
	case "join":
		argmax = 2
		f = &Join{}
	case "ksuid":
		f = &KSUIDToString{}
	case "log":
		f = &Log{}
	case "max":
		argmax = -1
		f = &reducer{fn: anymath.Max, name: name}
	case "min":
		argmax = -1
		f = &reducer{fn: anymath.Min, name: name}
	case "now":
		argmax = 0
		argmin = 0
		f = &Now{}
	case "round":
		f = &Round{}
	case "pow":
		argmin = 2
		argmax = 2
		f = &Pow{}
	case "sqrt":
		f = &Sqrt{}
	case "replace":
		argmin = 3
		argmax = 3
		f = &Replace{}
	case "rune_len":
		f = &RuneLen{}
	case "to_lower":
		f = &ToLower{}
	case "to_upper":
		f = &ToUpper{}
	case "trim":
		f = &Trim{}
	case "split":
		argmin = 2
		argmax = 2
		f = newSplit(zctx)
	case "trunc":
		argmin = 2
		argmax = 2
		f = &Trunc{}
	case "typename":
		argmax = 2
		f = &typeName{zctx: zctx}
	case "typeof":
		f = &TypeOf{zctx: zctx}
	case "typeunder":
		f = &typeUnder{zctx: zctx}
	case "nameof":
		f = &NameOf{}
	case "fields":
		f = NewFields(zctx)
	case "is":
		argmin = 1
		argmax = 2
		wantThis = true
		f = &Is{zctx: zctx}
	case "iserr":
		f = &IsErr{}
	case "to_base64":
		f = &ToBase64{}
	case "from_base64":
		f = &FromBase64{}
	case "to_hex":
		f = &ToHex{}
	case "from_hex":
		f = &FromHex{}
	case "network_of":
		argmax = 2
		f = &NetworkOf{}
	case "parse_uri":
		f = &ParseURI{marshaler: zson.NewZNGMarshalerWithContext(zctx)}
	case "parse_zson":
		f = &ParseZSON{zctx: zctx}
	case "quiet":
		f = &Quiet{}
	}
	if argmin != -1 && narg < argmin {
		return nil, false, ErrTooFewArgs
	}
	if argmax != -1 && narg > argmax {
		return nil, false, ErrTooManyArgs
	}
	return f, wantThis, nil
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

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#len
type LenFn struct{}

var _ Interface = (*LenFn)(nil)

func (l *LenFn) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	val := args[0]
	var length int
	switch typ := zed.AliasOf(args[0].Type).(type) {
	case *zed.TypeOfNull:
	case *zed.TypeRecord:
		length = len(typ.Columns)
	case *zed.TypeArray, *zed.TypeSet, *zed.TypeMap:
		var err error
		length, err = val.ContainerLength()
		if err != nil {
			panic(err)
		}
	case *zed.TypeOfBytes, *zed.TypeOfString, *zed.TypeOfBstring, *zed.TypeOfIP, *zed.TypeOfNet, *zed.TypeOfError:
		length = len(val.Bytes)
	default:
		return zed.NewErrorf("len: bad type: %s", zson.FormatType(typ))
	}
	return newInt64(ctx, int64(length))
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#typeof
type TypeOf struct {
	zctx *zed.Context
}

func (t *TypeOf) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	return ctx.CopyValue(*t.zctx.LookupTypeValue(args[0].Type))
}

type typeUnder struct {
	zctx *zed.Context
}

func (t *typeUnder) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	typ := zed.AliasOf(args[0].Type)
	return ctx.CopyValue(*t.zctx.LookupTypeValue(typ))
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#nameof
type NameOf struct{}

func (n *NameOf) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	typ := args[0].Type
	if alias, ok := typ.(*zed.TypeAlias); ok {
		return newString(ctx, alias.Name)
	}
	return zed.Missing
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#typename
type typeName struct {
	zctx *zed.Context
}

func (t *typeName) Call(ectx zed.Allocator, args []zed.Value) *zed.Value {
	if zed.AliasOf(args[0].Type) != zed.TypeString {
		return newErrorf(ectx, "typename: first argument not a string")
	}
	name := string(args[0].Bytes)
	if len(args) == 1 {
		typ := t.zctx.LookupTypeDef(name)
		if typ == nil {
			return zed.Missing
		}
		return t.zctx.LookupTypeValue(typ)
	}
	if zed.AliasOf(args[1].Type) != zed.TypeType {
		return newErrorf(ectx, "typename: second argument not a type value")
	}
	typ, err := t.zctx.LookupByValue(args[1].Bytes)
	if err != nil {
		return newError(ectx, err)
	}
	return t.zctx.LookupTypeValue(typ)
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#iserr
type IsErr struct{}

func (*IsErr) Call(_ zed.Allocator, args []zed.Value) *zed.Value {
	if args[0].IsError() {
		return zed.True
	}
	return zed.False
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#is
type Is struct {
	zctx *zed.Context
}

func (i *Is) Call(_ zed.Allocator, args []zed.Value) *zed.Value {
	zvSubject := args[0]
	zvTypeVal := args[1]
	if len(args) == 3 {
		zvSubject = args[1]
		zvTypeVal = args[2]
	}
	var typ zed.Type
	var err error
	if zvTypeVal.IsStringy() {
		typ, err = zson.ParseType(i.zctx, string(zvTypeVal.Bytes))
	} else {
		typ, err = i.zctx.LookupByValue(zvTypeVal.Bytes)
	}
	if err == nil && typ == zvSubject.Type {
		return zed.True
	}
	return zed.False
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#quiet
type Quiet struct{}

func (q *Quiet) Call(_ zed.Allocator, args []zed.Value) *zed.Value {
	val := args[0]
	if val.IsMissing() {
		return zed.Quiet
	}
	return &val
}

func newInt64(ctx zed.Allocator, native int64) *zed.Value {
	return newInt(ctx, zed.TypeInt64, native)
}

func newInt(ctx zed.Allocator, typ zed.Type, native int64) *zed.Value {
	//XXX we should have an interface to allocator where we can
	// append into some new bytes; for now, the byte slice goes through GC.
	return ctx.NewValue(typ, zed.EncodeInt(native))
}

func newUint64(ctx zed.Allocator, native uint64) *zed.Value {
	return newUint(ctx, zed.TypeUint64, native)
}

func newUint(ctx zed.Allocator, typ zed.Type, native uint64) *zed.Value {
	return ctx.NewValue(typ, zed.EncodeUint(native))
}

func newFloat64(ctx zed.Allocator, native float64) *zed.Value {
	return ctx.NewValue(zed.TypeFloat64, zed.EncodeFloat64(native))
}

func newDuration(ctx zed.Allocator, native nano.Duration) *zed.Value {
	return ctx.NewValue(zed.TypeDuration, zed.EncodeDuration(native))
}

func newTime(ctx zed.Allocator, native nano.Ts) *zed.Value {
	return ctx.NewValue(zed.TypeTime, zed.EncodeTime(native))
}

func newString(ctx zed.Allocator, native string) *zed.Value {
	return ctx.NewValue(zed.TypeString, zed.EncodeString(native))
}

func newError(ctx zed.Allocator, err error) *zed.Value {
	return zed.NewError(err)
}

func newErrorf(ctx zed.Allocator, format string, args ...interface{}) *zed.Value {
	return zed.NewErrorf(format, args...)
}

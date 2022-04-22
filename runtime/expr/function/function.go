package function

import (
	"errors"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/anymath"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/zson"
)

var (
	ErrBadArgument    = errors.New("bad argument")
	ErrNoSuchFunction = errors.New("no such function")
	ErrTooFewArgs     = errors.New("too few arguments")
	ErrTooManyArgs    = errors.New("too many arguments")
)

func New(zctx *zed.Context, name string, narg int) (expr.Function, field.Path, error) {
	argmin := 1
	argmax := 1
	var path field.Path
	var f expr.Function
	switch name {
	default:
		return nil, nil, ErrNoSuchFunction
	case "grep":
		// We special case grep here since a syntax error with the
		// special grep form will make it look like a function call
		// and we don't want the error to say unknown function.
		return nil, nil, errors.New("syntax error")
	case "len":
		f = &LenFn{zctx: zctx}
	case "abs":
		f = &Abs{zctx: zctx}
	case "every":
		path = field.New("ts")
		f = &Bucket{
			zctx: zctx,
			name: "every",
		}
	case "ceil":
		f = &Ceil{zctx: zctx}
	case "flatten":
		f = NewFlatten(zctx)
	case "floor":
		f = &Floor{zctx: zctx}
	case "join":
		argmax = 2
		f = &Join{zctx: zctx}
	case "ksuid":
		argmin = 0
		f = &KSUIDToString{zctx: zctx}
	case "log":
		f = &Log{zctx: zctx}
	case "max":
		argmax = -1
		f = &reducer{zctx: zctx, fn: anymath.Max, name: name}
	case "min":
		argmax = -1
		f = &reducer{zctx: zctx, fn: anymath.Min, name: name}
	case "now":
		argmax = 0
		argmin = 0
		f = &Now{}
	case "round":
		f = &Round{zctx: zctx}
	case "pow":
		argmin = 2
		argmax = 2
		f = &Pow{zctx: zctx}
	case "sqrt":
		f = &Sqrt{zctx: zctx}
	case "replace":
		argmin = 3
		argmax = 3
		f = &Replace{zctx: zctx}
	case "rune_len":
		f = &RuneLen{zctx: zctx}
	case "lower":
		f = &ToLower{zctx: zctx}
	case "upper":
		f = &ToUpper{zctx: zctx}
	case "trim":
		f = &Trim{zctx: zctx}
	case "split":
		argmin = 2
		argmax = 2
		f = newSplit(zctx)
	case "bucket":
		argmin = 2
		argmax = 2
		f = &Bucket{zctx: zctx}
	case "typename":
		argmax = 2
		f = &typeName{zctx: zctx}
	case "typeof":
		f = &TypeOf{zctx: zctx}
	case "typeunder":
		f = &typeUnder{zctx: zctx}
	case "nameof":
		f = &NameOf{zctx: zctx}
	case "fields":
		f = NewFields(zctx)
	case "has":
		argmax = -1
		f = &Has{}
	case "has_error":
		f = NewHasError()
	case "is":
		argmin = 1
		argmax = 2
		path = field.Path{}
		f = &Is{zctx: zctx}
	case "is_error":
		f = &IsErr{}
	case "error":
		f = &Error{zctx: zctx}
	case "kind":
		f = &Kind{zctx: zctx}
	case "base64":
		f = &Base64{zctx: zctx}
	case "hex":
		f = &Hex{zctx: zctx}
	case "cidr_match":
		argmin = 2
		argmax = 2
		f = &CIDRMatch{zctx: zctx}
	case "missing":
		argmax = -1
		f = &Missing{}
	case "network_of":
		argmax = 2
		f = &NetworkOf{zctx: zctx}
	case "nest_dotted":
		path = field.Path{}
		argmin = 0
		f = NewNestDotted(zctx)
	case "parse_uri":
		f = &ParseURI{zctx: zctx, marshaler: zson.NewZNGMarshalerWithContext(zctx)}
	case "parse_zson":
		f = &ParseZSON{zctx: zctx}
	case "quiet":
		f = &Quiet{zctx: zctx}
	case "under":
		f = &Under{zctx: zctx}
	case "unflatten":
		f = NewUnflatten(zctx)
	}
	if argmin != -1 && narg < argmin {
		return nil, nil, ErrTooFewArgs
	}
	if argmax != -1 && narg > argmax {
		return nil, nil, ErrTooManyArgs
	}
	return f, path, nil
}

// HasBoolResult returns true if the function name returns a Boolean value.
// XXX This is a hack so the semantic compiler can determine if a single call
// expr is a Filter or Put proc. At some point function declarations should have
// signatures so the return type can be introspected.
func HasBoolResult(name string) bool {
	switch name {
	case "has", "has_error", "is_error", "is", "missing", "cidr_match":
		return true
	}
	return false
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

func newFloat32(ctx zed.Allocator, native float32) *zed.Value {
	return ctx.NewValue(zed.TypeFloat32, zed.EncodeFloat32(native))
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

func newBytes(ctx zed.Allocator, bytes []byte) *zed.Value {
	return ctx.NewValue(zed.TypeBytes, bytes)
}

//XXX this should build the error in the allocator's memory but needs
// zctx for the type
func newError(zctx *zed.Context, ectx zed.Allocator, err error) *zed.Value {
	return zctx.NewError(err)
}

func newErrorf(zctx *zed.Context, ctx zed.Allocator, format string, args ...interface{}) *zed.Value {
	return zctx.NewErrorf(format, args...)
}

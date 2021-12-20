package function

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/anymath"
	"github.com/brimdata/zed/expr/result"
	"github.com/brimdata/zed/zson"
)

var (
	ErrBadArgument    = errors.New("bad argument")
	ErrNoSuchFunction = errors.New("no such function")
	ErrTooFewArgs     = errors.New("too few arguments")
	ErrTooManyArgs    = errors.New("too many arguments")
)

type Interface interface {
	Call([]zed.Value) *zed.Value
}

func New(zctx *zed.Context, name string, narg int) (Interface, bool, error) {
	argmin := 1
	argmax := 1
	var root bool
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
		f = &reducer{fn: anymath.Max}
	case "min":
		argmax = -1
		f = &reducer{fn: anymath.Min}
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
		root = true
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

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#len
type LenFn struct {
	stash result.Value
}

func (l *LenFn) Call(args []zed.Value) *zed.Value {
	zv := args[0]
	var length int
	switch typ := zed.AliasOf(args[0].Type).(type) {
	case *zed.TypeRecord:
		length = len(typ.Columns)
	case *zed.TypeArray, *zed.TypeSet, *zed.TypeMap:
		var err error
		length, err = zv.ContainerLength()
		if err != nil {
			panic(fmt.Errorf("len: corrupt Zed bytes: %w", err))
		}
	case *zed.TypeOfBytes, *zed.TypeOfString, *zed.TypeOfBstring, *zed.TypeOfIP, *zed.TypeOfNet, *zed.TypeOfError:
		length = len(zv.Bytes)
	default:
		err := fmt.Errorf("len: bad type: %s", zson.FormatType(typ))
		return l.stash.Error(err)
	}
	return l.stash.Int64(int64(length))
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#typeof
type TypeOf struct {
	zctx  *zed.Context
	stash zed.Value
}

func (t *TypeOf) Call(args []zed.Value) *zed.Value {
	t.stash = t.zctx.LookupTypeValue(args[0].Type)
	return &t.stash
}

type typeUnder struct {
	zctx  *zed.Context
	stash zed.Value
}

func (t *typeUnder) Call(args []zed.Value) *zed.Value {
	typ := zed.AliasOf(args[0].Type)
	t.stash = t.zctx.LookupTypeValue(typ)
	return &t.stash
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#nameof
type NameOf struct {
	stash zed.Value
}

func (n *NameOf) Call(args []zed.Value) *zed.Value {
	typ := args[0].Type
	if alias, ok := typ.(*zed.TypeAlias); ok {
		// XXX GC
		n.stash = zed.Value{zed.TypeString, zed.EncodeString(alias.Name)}
		return &n.stash
	}
	return zed.Missing
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#iserr
type IsErr struct{}

func (*IsErr) Call(args []zed.Value) *zed.Value {
	if args[0].IsError() {
		return zed.True
	}
	return zed.False
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#is
type Is struct {
	zctx *zed.Context
}

func (i *Is) Call(args []zed.Value) *zed.Value {
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

//XXX need to add doc
// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#quiet
type Quiet struct {
	zctx *zed.Context
}

func (q *Quiet) Call(args []zed.Value) *zed.Value {
	val := args[0]
	if val.IsMissing() {
		return zed.Quiet
	}
	return &val
}

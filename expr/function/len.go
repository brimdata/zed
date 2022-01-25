package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zson"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#len
type LenFn struct {
	zctx *zed.Context
}

var _ Interface = (*LenFn)(nil)

func (l *LenFn) Call(ectx zed.Allocator, args []zed.Value) *zed.Value {
	val := args[0]
	var length int
	switch typ := zed.TypeUnder(args[0].Type).(type) {
	case *zed.TypeOfNull:
	case *zed.TypeRecord:
		length = len(typ.Columns)
	case *zed.TypeArray, *zed.TypeSet, *zed.TypeMap:
		var err error
		length, err = val.ContainerLength()
		if err != nil {
			panic(err)
		}
	case *zed.TypeOfBytes, *zed.TypeOfString, *zed.TypeOfIP, *zed.TypeOfNet:
		length = len(val.Bytes)
	case *zed.TypeError:
		return l.zctx.WrapError("len()", &val)
	case *zed.TypeOfType:
		t, err := l.zctx.LookupByValue(val.Bytes)
		if err != nil {
			return newError(l.zctx, ectx, err)
		}
		length = typeLength(t)
	default:
		return l.zctx.NewErrorf("len: bad type: %s", zson.FormatType(typ))
	}
	return newInt64(ectx, int64(length))
}

func typeLength(typ zed.Type) int {
	switch typ := typ.(type) {
	case *zed.TypeNamed:
		return typeLength(typ.Type)
	case *zed.TypeRecord:
		return len(typ.Columns)
	case *zed.TypeUnion:
		return len(typ.Types)
	case *zed.TypeSet:
		return typeLength(typ.Type)
	case *zed.TypeArray:
		return typeLength(typ.Type)
	case *zed.TypeEnum:
		return len(typ.Symbols)
	case *zed.TypeMap:
		return typeLength(typ.ValType)
	case *zed.TypeError:
		return typeLength(typ.Type)
	default:
		// Primitive type
		return 1
	}
}

package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/sam/expr"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#len
type LenFn struct {
	zctx *zed.Context
}

func (l *LenFn) Call(ectx expr.Context, args []zed.Value) zed.Value {
	val := args[0].Under(ectx.Arena())
	var length int
	switch typ := zed.TypeUnder(val.Type()).(type) {
	case *zed.TypeOfNull:
	case *zed.TypeRecord:
		length = len(typ.Fields)
	case *zed.TypeArray, *zed.TypeSet, *zed.TypeMap:
		var err error
		length, err = val.ContainerLength()
		if err != nil {
			panic(err)
		}
	case *zed.TypeOfBytes, *zed.TypeOfString, *zed.TypeOfIP, *zed.TypeOfNet:
		length = len(val.Bytes())
	case *zed.TypeError:
		return l.zctx.WrapError(ectx.Arena(), "len()", val)
	case *zed.TypeOfType:
		t, err := l.zctx.LookupByValue(val.Bytes())
		if err != nil {
			return l.zctx.NewError(ectx.Arena(), err)
		}
		length = TypeLength(t)
	default:
		return l.zctx.WrapError(ectx.Arena(), "len: bad type", val)
	}
	return zed.NewInt64(int64(length))
}

func TypeLength(typ zed.Type) int {
	switch typ := typ.(type) {
	case *zed.TypeNamed:
		return TypeLength(typ.Type)
	case *zed.TypeRecord:
		return len(typ.Fields)
	case *zed.TypeUnion:
		return len(typ.Types)
	case *zed.TypeSet:
		return TypeLength(typ.Type)
	case *zed.TypeArray:
		return TypeLength(typ.Type)
	case *zed.TypeEnum:
		return len(typ.Symbols)
	case *zed.TypeMap:
		return TypeLength(typ.ValType)
	case *zed.TypeError:
		return TypeLength(typ.Type)
	default:
		// Primitive type
		return 1
	}
}

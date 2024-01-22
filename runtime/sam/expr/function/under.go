package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/expr"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#under
type Under struct {
	zctx *zed.Context
}

func (u *Under) Call(ectx expr.Context, args []zed.Value) zed.Value {
	val := args[0]
	switch typ := args[0].Type().(type) {
	case *zed.TypeNamed:
		return ectx.Arena().NewValue(typ.Type, val.Bytes())
	case *zed.TypeError:
		return ectx.Arena().NewValue(typ.Type, val.Bytes())
	case *zed.TypeUnion:
		return ectx.Arena().NewValue(typ.Untag(val.Bytes()))
	case *zed.TypeOfType:
		t, err := u.zctx.LookupByValue(val.Bytes())
		if err != nil {
			return ectx.Arena().NewError(err)
		}
		return ectx.Arena().LookupTypeValue(zed.TypeUnder(t))
	default:
		return val
	}
}

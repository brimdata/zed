package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/sam/expr"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#under
type Under struct {
	zctx *zed.Context
}

func (u *Under) Call(ectx expr.Context, args []zed.Value) zed.Value {
	val := args[0]
	switch typ := args[0].Type().(type) {
	case *zed.TypeNamed:
		return ectx.Arena().New(typ.Type, val.Bytes())
	case *zed.TypeError:
		return ectx.Arena().New(typ.Type, val.Bytes())
	case *zed.TypeUnion:
		return ectx.Arena().New(typ.Untag(val.Bytes()))
	case *zed.TypeOfType:
		t, err := u.zctx.LookupByValue(val.Bytes())
		if err != nil {
			return u.zctx.NewError(ectx.Arena(), err)
		}
		return u.zctx.LookupTypeValue(ectx.Arena(), zed.TypeUnder(t))
	default:
		return val
	}
}

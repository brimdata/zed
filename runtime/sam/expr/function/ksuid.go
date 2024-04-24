package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/sam/expr"
	"github.com/segmentio/ksuid"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#ksuid
type KSUIDToString struct {
	zctx *zed.Context
}

func (k *KSUIDToString) Call(ectx expr.Context, args []zed.Value) zed.Value {
	if len(args) == 0 {
		return ectx.Arena().NewBytes(ksuid.New().Bytes())
	}
	val := args[0]
	switch val.Type().ID() {
	case zed.IDBytes:
		if val.IsNull() {
			return k.zctx.NewErrorf(ectx.Arena(), "ksuid: illegal null argument")
		}
		// XXX GC
		id, err := ksuid.FromBytes(val.Bytes())
		if err != nil {
			panic(err)
		}
		return ectx.Arena().NewString(id.String())
	case zed.IDString:
		// XXX GC
		id, err := ksuid.Parse(string(val.Bytes()))
		if err != nil {
			return k.zctx.WrapError(ectx.Arena(), "ksuid: "+err.Error(), val)
		}
		return ectx.Arena().NewBytes(id.Bytes())
	default:
		return k.zctx.WrapError(ectx.Arena(), "ksuid: argument must a bytes or string type", val)
	}
}

package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/segmentio/ksuid"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#ksuid
type KSUIDToString struct {
	zctx *zed.Context
}

func (k *KSUIDToString) Call(ectx expr.Context, args []zed.Value) zed.Value {
	arena := ectx.Arena()
	if len(args) == 0 {
		return arena.NewBytes(ksuid.New().Bytes())
	}
	val := args[0]
	switch val.Type().ID() {
	case zed.IDBytes:
		if val.IsNull() {
			return arena.NewErrorf("ksuid: illegal null argument")
		}
		// XXX GC
		id, err := ksuid.FromBytes(val.Bytes())
		if err != nil {
			panic(err)
		}
		return arena.NewString(id.String())
	case zed.IDString:
		// XXX GC
		id, err := ksuid.Parse(string(val.Bytes()))
		if err != nil {
			return arena.WrapError("ksuid: "+err.Error(), val)
		}
		return arena.NewBytes(id.Bytes())
	default:
		return arena.WrapError("ksuid: argument must a bytes or string type", val)
	}
}

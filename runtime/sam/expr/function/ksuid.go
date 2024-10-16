package function

import (
	"github.com/brimdata/super"
	"github.com/segmentio/ksuid"
)

// https://github.com/brimdata/super/blob/main/docs/language/functions.md#ksuid
type KSUIDToString struct {
	zctx *zed.Context
}

func (k *KSUIDToString) Call(_ zed.Allocator, args []zed.Value) zed.Value {
	if len(args) == 0 {
		return zed.NewBytes(ksuid.New().Bytes())
	}
	val := args[0]
	switch val.Type().ID() {
	case zed.IDBytes:
		if val.IsNull() {
			return k.zctx.NewErrorf("ksuid: illegal null argument")
		}
		// XXX GC
		id, err := ksuid.FromBytes(val.Bytes())
		if err != nil {
			panic(err)
		}
		return zed.NewString(id.String())
	case zed.IDString:
		// XXX GC
		id, err := ksuid.Parse(string(val.Bytes()))
		if err != nil {
			return k.zctx.WrapError("ksuid: "+err.Error(), val)
		}
		return zed.NewBytes(id.Bytes())
	default:
		return k.zctx.WrapError("ksuid: argument must a bytes or string type", val)
	}
}

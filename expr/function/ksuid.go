package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#ksuid
type KSUIDToString struct {
	zctx *zed.Context
}

func (k *KSUIDToString) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	if len(args) == 0 {
		return newBytes(ctx, ksuid.New().Bytes())
	}
	zv := args[0]
	switch zv.Type.ID() {
	case zed.IDBytes:
		if zv.Bytes == nil {
			return newErrorf(k.zctx, ctx, "ksuid: illegal null argument")
		}
		// XXX GC
		id, err := ksuid.FromBytes(zv.Bytes)
		if err != nil {
			panic(err)
		}
		return newString(ctx, id.String())
	case zed.IDString:
		// XXX GC
		id, err := ksuid.Parse(string(zv.Bytes))
		if err != nil {
			return newErrorf(k.zctx, ctx, "ksuid: %s (bad argument: %s)", err, zson.String(zv))
		}
		return newBytes(ctx, id.Bytes())
	default:
		return newErrorf(k.zctx, ctx, "ksuid: argument must a bytes or string type (bad argument: %s)", zson.String(zv))
	}
}

package function

import (
	"github.com/brimdata/zed"
	"github.com/segmentio/ksuid"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#ksuid
type KSUIDToString struct {
	zctx *zed.Context
}

func (k *KSUIDToString) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	zv := args[0]
	if zv.Type.ID() != zed.IDBytes {
		return newErrorf(k.zctx, ctx, "ksuid: not a bytes type")
	}
	// XXX GC
	id, err := ksuid.FromBytes(zv.Bytes)
	if err != nil {
		panic(err)
	}
	return newString(ctx, id.String())
}

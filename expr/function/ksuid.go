package function

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/segmentio/ksuid"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#ksuid
type KSUIDToString struct {
	stash zed.Value
}

func (k *KSUIDToString) Call(args []zed.Value) *zed.Value {
	zv := args[0]
	if zv.Type.ID() != zed.IDBytes {
		k.stash = zed.NewErrorf("not a bytes type")
		return &k.stash
	}
	// XXX GC
	id, err := ksuid.FromBytes(zv.Bytes)
	if err != nil {
		panic(fmt.Errorf("ksuid: corrupt Zed bytes", err))
	}
	k.stash = zed.Value{zed.TypeString, zed.EncodeString(id.String())}
	return &k.stash
}

package function

import (
	"github.com/brimdata/zed"
	"github.com/segmentio/ksuid"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#ksuid
type KSUIDToString struct{}

func (*KSUIDToString) Call(args []zed.Value) (zed.Value, error) {
	zv := args[0]
	if zv.Type.ID() != zed.IDBytes {
		return zed.NewErrorf("not a bytes type"), nil
	}
	// XXX GC
	id, err := ksuid.FromBytes(zv.Bytes)
	if err != nil {
		return zed.NewErrorf("error parsing bytes as ksuid: %s", err), nil
	}
	return zed.Value{zed.TypeString, zed.EncodeString(id.String())}, nil
}

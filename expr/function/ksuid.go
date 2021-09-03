package function

import (
	"github.com/brimdata/zed/expr/result"
	"github.com/brimdata/zed/zng"
	"github.com/segmentio/ksuid"
)

type ksuidToString struct {
	result.Buffer
}

func (k *ksuidToString) Call(args []zng.Value) (zng.Value, error) {
	zv := args[0]
	if zv.Type.ID() != zng.IDBytes {
		return zng.NewErrorf("not a bytes type"), nil
	}
	// XXX GC
	id, err := ksuid.FromBytes(zv.Bytes)
	if err != nil {
		return zng.NewErrorf("error parsing bytes as ksuid: %s", err), nil
	}
	return zng.Value{zng.TypeString, zng.EncodeString(id.String())}, nil
}

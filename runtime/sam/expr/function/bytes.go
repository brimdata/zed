package function

import (
	"encoding/base64"
	"encoding/hex"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/sam/expr"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#base64
type Base64 struct {
	zctx *zed.Context
}

func (b *Base64) Call(ectx expr.Context, args []zed.Value) zed.Value {
	arena := ectx.Arena()
	val := args[0]
	switch val.Type().ID() {
	case zed.IDBytes:
		if val.IsNull() {
			return arena.NewErrorf("base64: illegal null argument")
		}
		return arena.NewString(base64.StdEncoding.EncodeToString(val.Bytes()))
	case zed.IDString:
		if val.IsNull() {
			return zed.Null
		}
		bytes, err := base64.StdEncoding.DecodeString(zed.DecodeString(val.Bytes()))
		if err != nil {
			return arena.WrapError("base64: string argument is not base64", val)
		}
		return arena.NewBytes(bytes)
	default:
		return arena.WrapError("base64: argument must a bytes or string type", val)
	}
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#hex
type Hex struct {
	zctx *zed.Context
}

func (h *Hex) Call(ectx expr.Context, args []zed.Value) zed.Value {
	arena := ectx.Arena()
	val := args[0]
	switch val.Type().ID() {
	case zed.IDBytes:
		if val.IsNull() {
			return arena.NewErrorf("hex: illegal null argument")
		}
		return arena.NewString(hex.EncodeToString(val.Bytes()))
	case zed.IDString:
		if val.IsNull() {
			return zed.NullString
		}
		b, err := hex.DecodeString(zed.DecodeString(val.Bytes()))
		if err != nil {
			return arena.WrapError("hex: string argument is not hexidecimal", val)
		}
		return arena.NewBytes(b)
	default:
		return arena.WrapError("base64: argument must a bytes or string type", val)
	}
}

package function

import (
	"encoding/base64"
	"encoding/hex"

	"github.com/brimdata/zed"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#base64
type Base64 struct {
	zctx *zed.Context
}

func (b *Base64) Call(_ zed.Allocator, args []zed.Value) zed.Value {
	val := args[0].Under()
	switch val.Type().ID() {
	case zed.IDBytes:
		if val.IsNull() {
			return b.zctx.NewErrorf("base64: illegal null argument")
		}
		return zed.NewString(base64.StdEncoding.EncodeToString(val.Bytes()))
	case zed.IDString:
		if val.IsNull() {
			return zed.NullBytes
		}
		bytes, err := base64.StdEncoding.DecodeString(zed.DecodeString(val.Bytes()))
		if err != nil {
			return b.zctx.WrapError("base64: string argument is not base64", val)
		}
		return zed.NewBytes(bytes)
	default:
		return b.zctx.WrapError("base64: argument must a bytes or string type", val)
	}
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#hex
type Hex struct {
	zctx *zed.Context
}

func (h *Hex) Call(_ zed.Allocator, args []zed.Value) zed.Value {
	val := args[0].Under()
	switch val.Type().ID() {
	case zed.IDBytes:
		if val.IsNull() {
			return h.zctx.NewErrorf("hex: illegal null argument")
		}
		return zed.NewString(hex.EncodeToString(val.Bytes()))
	case zed.IDString:
		if val.IsNull() {
			return zed.NullBytes
		}
		b, err := hex.DecodeString(zed.DecodeString(val.Bytes()))
		if err != nil {
			return h.zctx.WrapError("hex: string argument is not hexidecimal", val)
		}
		return zed.NewBytes(b)
	default:
		return h.zctx.WrapError("base64: argument must a bytes or string type", val)
	}
}

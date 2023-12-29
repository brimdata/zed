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

func (b *Base64) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	val := args[0]
	switch val.Type().ID() {
	case zed.IDBytes:
		if val.IsNull() {
			return newErrorf(b.zctx, ctx, "base64: illegal null argument")
		}
		return newString(ctx, base64.StdEncoding.EncodeToString(val.Bytes()))
	case zed.IDString:
		if val.IsNull() {
			return zed.Null
		}
		bytes, err := base64.StdEncoding.DecodeString(zed.DecodeString(val.Bytes()))
		if err != nil {
			return wrapError(b.zctx, ctx, "base64: string argument is not base64", &val)
		}
		return newBytes(ctx, bytes)
	default:
		return wrapError(b.zctx, ctx, "base64: argument must a bytes or string type", &val)
	}
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#hex
type Hex struct {
	zctx *zed.Context
}

func (h *Hex) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	val := args[0]
	switch val.Type().ID() {
	case zed.IDBytes:
		if val.IsNull() {
			return newErrorf(h.zctx, ctx, "hex: illegal null argument")
		}
		return newString(ctx, hex.EncodeToString(val.Bytes()))
	case zed.IDString:
		if val.IsNull() {
			return zed.NullString
		}
		b, err := hex.DecodeString(zed.DecodeString(val.Bytes()))
		if err != nil {
			return wrapError(h.zctx, ctx, "hex: string argument is not hexidecimal", &val)
		}
		return newBytes(ctx, b)
	default:
		return wrapError(h.zctx, ctx, "base64: argument must a bytes or string type", &val)
	}
}

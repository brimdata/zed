package function

import (
	"encoding/base64"
	"encoding/hex"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zson"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#base64
type Base64 struct {
	zctx *zed.Context
}

func (b *Base64) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	zv := args[0]
	switch zv.Type.ID() {
	case zed.IDBytes:
		if zv.Bytes == nil {
			return newErrorf(b.zctx, ctx, "base64: illegal null argument")
		}
		return newString(ctx, base64.StdEncoding.EncodeToString(zv.Bytes))
	case zed.IDString:
		if zv.Bytes == nil {
			return zed.Null
		}
		bytes, err := base64.StdEncoding.DecodeString(zed.DecodeString(zv.Bytes))
		if err != nil {
			return newErrorf(b.zctx, ctx, "base64: string argument is not base64: %q", string(zv.Bytes))
		}
		return newBytes(ctx, bytes)
	default:
		return newErrorf(b.zctx, ctx, "base64: argument must a bytes or string type (bad argument: %s)", zson.String(zv))
	}
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#hex
type Hex struct {
	zctx *zed.Context
}

func (h *Hex) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	val := args[0]
	switch val.Type.ID() {
	case zed.IDBytes:
		if val.Bytes == nil {
			return newErrorf(h.zctx, ctx, "hex: illegal null argument")
		}
		return newString(ctx, hex.EncodeToString(val.Bytes))
	case zed.IDString:
		if val.Bytes == nil {
			return zed.NullString
		}
		b, err := hex.DecodeString(zed.DecodeString(val.Bytes))
		if err != nil {
			return newErrorf(h.zctx, ctx, "hex: string argument is not hexidecimal: %q", string(val.Bytes))
		}
		return newBytes(ctx, b)
	default:
		return newErrorf(h.zctx, ctx, "base64: argument must a bytes or string type (bad argument: %s)", zson.String(val))
	}
}

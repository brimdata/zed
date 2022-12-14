package function

import (
	"errors"
	"net/netip"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#network_of
type NetworkOf struct {
	zctx *zed.Context
}

func (n *NetworkOf) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	id := args[0].Type.ID()
	if id != zed.IDIP {
		return newErrorf(n.zctx, ctx, "network_of: not an IP")
	}
	ip := zed.DecodeIP(args[0].Bytes)
	var bits int
	if len(args) == 1 {
		switch {
		case !ip.Is4():
			return newErrorf(n.zctx, ctx, "network_of: not an IPv4 address")
		case ip.As4()[0] < 0x80:
			bits = 8
		case ip.As4()[0] < 0xc0:
			bits = 16
		default:
			bits = 24
		}
	} else {
		// two args
		body := args[1].Bytes
		switch id := args[1].Type.ID(); {
		case id == zed.IDIP:
			mask := zed.DecodeIP(body)
			if mask.BitLen() != ip.BitLen() {
				return newErrorf(n.zctx, ctx, "network_of: address %s and mask %s have different lengths", ip, mask)
			}
			bits = zed.LeadingOnes(mask.AsSlice())
			if netip.PrefixFrom(mask, bits).Masked().Addr() != mask {
				return newErrorf(n.zctx, ctx, "network_of: mask %s is non-contiguous", mask)
			}
		case zed.IsInteger(id):
			if zed.IsSigned(id) {
				bits = int(zed.DecodeInt(body))
			} else {
				bits = int(zed.DecodeUint(body))
			}
			if bits > 128 || bits > 32 && ip.Is4() {
				return newErrorf(n.zctx, ctx, "network_of: cidr bit count out of range")
			}
		default:
			return newErrorf(n.zctx, ctx, "network_of: bad arg for cidr mask")
		}
	}
	// Mask for canonical form.
	prefix := netip.PrefixFrom(ip, bits).Masked()
	return ctx.NewValue(zed.TypeNet, zed.EncodeNet(prefix))
}

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#cidr_match
type CIDRMatch struct {
	zctx *zed.Context
}

var errMatch = errors.New("match")

func (c *CIDRMatch) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	maskVal := args[0]
	if maskVal.Type.ID() != zed.IDNet {
		return newErrorf(c.zctx, ctx, "cidr_match: not a net: %s", zson.String(maskVal))
	}
	prefix := zed.DecodeNet(maskVal.Bytes)
	if errMatch == args[1].Walk(func(typ zed.Type, body zcode.Bytes) error {
		if typ.ID() == zed.IDIP {
			if prefix.Contains(zed.DecodeIP(body)) {
				return errMatch
			}
		}
		return nil
	}) {
		return zed.True
	}
	return zed.False
}

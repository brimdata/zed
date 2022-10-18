package function

import (
	"bytes"
	"errors"
	"net"

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
	// XXX GC
	ip := zed.DecodeIP(args[0].Bytes)
	var mask net.IPMask
	if len(args) == 1 {
		mask = net.IP(ip.AsSlice()).DefaultMask()
		if mask == nil {
			return newErrorf(n.zctx, ctx, "network_of: not an IPv4 address")
		}
	} else {
		// two args
		id := args[1].Type.ID()
		body := args[1].Bytes
		if id == zed.IDNet {
			cidrMask := zed.DecodeNet(body)
			if !bytes.Equal(cidrMask.IP, cidrMask.Mask) {
				return newErrorf(n.zctx, ctx, "network_of: network arg not a cidr mask")
			}
			mask = cidrMask.Mask
		} else if zed.IsInteger(id) {
			var nbits uint
			if zed.IsSigned(id) {
				nbits = uint(zed.DecodeInt(body))
			} else {
				nbits = uint(zed.DecodeUint(body))
			}
			if nbits > 64 {
				return newErrorf(n.zctx, ctx, "network_of: cidr bit count out of range")
			}
			mask = net.CIDRMask(int(nbits), int(ip.BitLen()))
		} else {
			return newErrorf(n.zctx, ctx, "network_of: bad arg for cidr mask")
		}
	}
	// XXX GC
	netIP := net.IP(ip.AsSlice()).Mask(mask)
	v := &net.IPNet{IP: netIP, Mask: mask}
	return ctx.NewValue(zed.TypeNet, zed.EncodeNet(v))
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
	cidrMask := zed.DecodeNet(maskVal.Bytes)
	if errMatch == args[1].Walk(func(typ zed.Type, body zcode.Bytes) error {
		if typ.ID() == zed.IDIP {
			addr := net.IP(zed.DecodeIP(body).AsSlice())
			if cidrMask.IP.Equal(addr.Mask(cidrMask.Mask)) {
				return errMatch
			}
		}
		return nil
	}) {
		return zed.True
	}
	return zed.False
}

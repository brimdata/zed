package function

import (
	"bytes"
	"net"

	"github.com/brimdata/zed"
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
		mask = ip.IPAddr().IP.DefaultMask()
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
	netIP := ip.IPAddr().IP.Mask(mask)
	v := &net.IPNet{netIP, mask}
	return ctx.NewValue(zed.TypeNet, zed.EncodeNet(v))
}

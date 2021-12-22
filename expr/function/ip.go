package function

import (
	"bytes"
	"net"

	"github.com/brimdata/zed"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#network_of
type NetworkOf struct{}

func (n *NetworkOf) Call(ctx zed.Allocator, args []zed.Value) *zed.Value {
	id := args[0].Type.ID()
	if id != zed.IDIP {
		return newErrorf(ctx, "network_of: not an IP")
	}
	// XXX GC
	ip, err := zed.DecodeIP(args[0].Bytes)
	if err != nil {
		panic(err)
	}
	var mask net.IPMask
	if len(args) == 1 {
		mask = ip.DefaultMask()
		if mask == nil {
			return newErrorf(ctx, "network_of: not an IPv4 address")
		}
	} else {
		// two args
		id := args[1].Type.ID()
		body := args[1].Bytes
		if id == zed.IDNet {
			var err error
			cidrMask, err := zed.DecodeNet(body)
			if err != nil {
				panic(err)
			}
			if !bytes.Equal(cidrMask.IP, cidrMask.Mask) {
				return newErrorf(ctx, "network_of: network arg not a cidr mask")
			}
			mask = cidrMask.Mask
		} else if zed.IsInteger(id) {
			var nbits uint
			if zed.IsSigned(id) {
				v, err := zed.DecodeInt(body)
				if err != nil {
					panic(err)
				}
				nbits = uint(v)
			} else {
				v, err := zed.DecodeUint(body)
				if err != nil {
					panic(err)
				}
				nbits = uint(v)
			}
			if nbits > 64 {
				return newErrorf(ctx, "network_of: cidr bit count out of range")
			}
			mask = net.CIDRMask(int(nbits), 8*len(ip))
		} else {
			return newErrorf(ctx, "network_of: bad arg for cidr mask")
		}
	}
	// XXX GC
	netIP := ip.Mask(mask)
	v := &net.IPNet{netIP, mask}
	return ctx.NewValue(zed.TypeNet, zed.EncodeNet(v))
}

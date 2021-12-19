package function

import (
	"bytes"
	"fmt"
	"net"

	"github.com/brimdata/zed"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#network_of
type NetworkOf struct {
	stash zed.Value
}

func (n *NetworkOf) Call(args []zed.Value) *zed.Value {
	id := args[0].Type.ID()
	if id != zed.IDIP {
		n.stash = zed.NewErrorf("network_of: not an IP")
		return &n.stash
	}
	// XXX GC
	ip, err := zed.DecodeIP(args[0].Bytes)
	if err != nil {
		panic(fmt.Errorf("network_of: corrupt Zed bytes: %w", err))
	}
	var mask net.IPMask
	if len(args) == 1 {
		mask = ip.DefaultMask()
		if mask == nil {
			n.stash = zed.NewErrorf("network_of: not an IP")
			return &n.stash
		}
	} else {
		// two args
		id := args[1].Type.ID()
		body := args[1].Bytes
		if id == zed.IDNet {
			var err error
			cidrMask, err := zed.DecodeNet(body)
			if err != nil {
				panic(fmt.Errorf("network_of: corrupt Zed bytes: %w", err))
			}
			if !bytes.Equal(cidrMask.IP, cidrMask.Mask) {
				n.stash = zed.NewErrorf("network_of: network arg not a cidr mask")
				return &n.stash
			}
			mask = cidrMask.Mask
		} else if zed.IsInteger(id) {
			var nbits uint
			if zed.IsSigned(id) {
				v, err := zed.DecodeInt(body)
				if err != nil {
					panic(fmt.Errorf("network_of: corrupt Zed bytes: %w", err))
				}
				nbits = uint(v)
			} else {
				v, err := zed.DecodeUint(body)
				if err != nil {
					panic(fmt.Errorf("network_of: corrupt Zed bytes: %w", err))
				}
				nbits = uint(v)
			}
			if nbits > 64 {
				n.stash = zed.NewErrorf("network_of: cidr bit count out of range")
				return &n.stash
			}
			mask = net.CIDRMask(int(nbits), 8*len(ip))
		} else {
			n.stash = zed.NewErrorf("network_of: bad arg for cidr mask")
			return &n.stash
		}
	}
	// XXX GC
	netIP := ip.Mask(mask)
	v := &net.IPNet{netIP, mask}
	n.stash = zed.Value{zed.TypeNet, zed.EncodeNet(v)}
	return &n.stash
}

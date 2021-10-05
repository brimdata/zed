package function

import (
	"bytes"
	"net"

	"github.com/brimdata/zed"
)

type networkOf struct{}

func (*networkOf) Call(args []zed.Value) (zed.Value, error) {
	id := args[0].Type.ID()
	if id != zed.IDIP {
		return zed.NewErrorf("not an IP"), nil
	}
	// XXX GC
	ip, err := zed.DecodeIP(args[0].Bytes)
	if err != nil {
		return zed.NewError(err), nil
	}
	var mask net.IPMask
	if len(args) == 1 {
		mask = ip.DefaultMask()
		if mask == nil {
			return zed.NewErrorf("not an IPv4"), nil
		}
	} else {
		// two args
		id := args[1].Type.ID()
		body := args[1].Bytes
		if id == zed.IDNet {
			var err error
			cidrMask, err := zed.DecodeNet(body)
			if err != nil {
				return zed.NewError(err), nil
			}
			if !bytes.Equal(cidrMask.IP, cidrMask.Mask) {
				return zed.NewErrorf("network arg not a cidr mask"), nil
			}
			mask = cidrMask.Mask
		} else if zed.IsInteger(id) {
			var nbits uint
			if zed.IsSigned(id) {
				v, err := zed.DecodeInt(body)
				if err != nil {
					return zed.NewError(err), nil
				}
				nbits = uint(v)
			} else {
				v, err := zed.DecodeUint(body)
				if err != nil {
					return zed.NewError(err), nil
				}
				nbits = uint(v)
			}
			if nbits > 64 {
				return zed.NewErrorf("cidr bit count out of range"), nil
			}
			mask = net.CIDRMask(int(nbits), 8*len(ip))
		} else {
			return zed.NewErrorf("bad arg for cidr mask"), nil
		}
	}
	// XXX GC
	netIP := ip.Mask(mask)
	v := &net.IPNet{netIP, mask}
	return zed.Value{zed.TypeNet, zed.EncodeNet(v)}, nil
}

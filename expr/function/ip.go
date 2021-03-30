package function

import (
	"bytes"
	"net"

	"github.com/brimdata/zed/expr/result"
	"github.com/brimdata/zed/zng"
)

type networkOf struct {
	result.Buffer
}

func (n *networkOf) Call(args []zng.Value) (zng.Value, error) {
	id := args[0].Type.ID()
	if id != zng.IdIP {
		return zng.NewErrorf("not an IP"), nil
	}
	// XXX GC
	ip, err := zng.DecodeIP(args[0].Bytes)
	if err != nil {
		return zng.NewError(err), nil
	}
	var mask net.IPMask
	if len(args) == 1 {
		mask = ip.DefaultMask()
		if mask == nil {
			return zng.NewErrorf("not an IPv4"), nil
		}
	} else {
		// two args
		id := args[1].Type.ID()
		body := args[1].Bytes
		if id == zng.IdNet {
			var err error
			cidrMask, err := zng.DecodeNet(body)
			if err != nil {
				return zng.NewError(err), nil
			}
			if !bytes.Equal(cidrMask.IP, cidrMask.Mask) {
				return zng.NewErrorf("network arg not a cidr mask"), nil
			}
			mask = cidrMask.Mask
		} else if zng.IsInteger(id) {
			var nbits uint
			if zng.IsSigned(id) {
				v, err := zng.DecodeInt(body)
				if err != nil {
					return zng.NewError(err), nil
				}
				nbits = uint(v)
			} else {
				v, err := zng.DecodeUint(body)
				if err != nil {
					return zng.NewError(err), nil
				}
				nbits = uint(v)
			}
			if nbits > 64 {
				return zng.NewErrorf("cidr bit count out of range"), nil
			}
			mask = net.CIDRMask(int(nbits), 8*len(ip))
		} else {
			return zng.NewErrorf("bad arg for cidr mask"), nil
		}
	}
	// XXX GC
	netIP := ip.Mask(mask)
	v := &net.IPNet{netIP, mask}
	return zng.Value{zng.TypeNet, zng.EncodeNet(v)}, nil
}

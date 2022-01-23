package zed

import (
	"net"

	"github.com/brimdata/zed/zcode"
)

type TypeOfNet struct{}

func NewNet(s *net.IPNet) *Value {
	return &Value{TypeNet, EncodeNet(s)}
}

func AppendNet(zb zcode.Bytes, subnet *net.IPNet) zcode.Bytes {
	if ip := subnet.IP.To4(); ip != nil {
		zb = append(zb, ip...)
		if len(subnet.Mask) == 16 {
			return append(zb, subnet.Mask[12:]...)
		}
		return append(zb, subnet.Mask...)
	}
	zb = append(zb, subnet.IP...)
	return append(zb, subnet.Mask...)
}

func EncodeNet(subnet *net.IPNet) zcode.Bytes {
	return AppendNet(nil, subnet)
}

func DecodeNet(zv zcode.Bytes) *net.IPNet {
	if zv == nil {
		return nil
	}
	switch len(zv) {
	case 8:
		return &net.IPNet{
			IP:   net.IP(zv[:4]),
			Mask: net.IPMask(zv[4:]),
		}
	case 32:
		return &net.IPNet{
			IP:   net.IP(zv[:16]),
			Mask: net.IPMask(zv[16:]),
		}
	}
	panic("failure trying to decode IP subnet that is not 8 or 32 bytes long")
}

func (t *TypeOfNet) ID() int {
	return IDNet
}

func (t *TypeOfNet) String() string {
	return "net"
}

func (t *TypeOfNet) Format(zb zcode.Bytes) string {
	return DecodeNet(zb).String()
}

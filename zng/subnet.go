package zng

import (
	"errors"
	"net"

	"github.com/mccanne/zq/zcode"
)

type TypeOfSubnet struct{}

func NewSubnet(s *net.IPNet) Value {
	return Value{TypeSubnet, EncodeSubnet(s)}
}

func EncodeSubnet(subnet *net.IPNet) zcode.Bytes {
	var b [32]byte
	ip := subnet.IP.To4()
	if ip != nil {
		copy(b[:], ip)
		if len(subnet.Mask) == 16 {
			copy(b[4:], subnet.Mask[12:])
		} else {
			copy(b[4:], subnet.Mask)
		}
		return b[:8]
	}
	copy(b[:], ip)
	copy(b[16:], subnet.Mask)
	return b[:]
}

func DecodeSubnet(zv zcode.Bytes) (*net.IPNet, error) {
	if zv == nil {
		return nil, ErrUnset
	}
	switch len(zv) {
	case 8:
		ip := net.IP(zv[:4])
		mask := net.IPMask(zv[4:])
		return &net.IPNet{
			IP:   ip,
			Mask: mask,
		}, nil
	case 32:
		ip := net.IP(zv[:16])
		mask := net.IPMask(zv[16:])
		return &net.IPNet{
			IP:   ip,
			Mask: mask,
		}, nil
	}
	return nil, errors.New("failure trying to decode IP subnet that is not 8 or 32 bytes long")
}

func (t *TypeOfSubnet) Parse(in []byte) (zcode.Bytes, error) {
	_, subnet, err := net.ParseCIDR(string(in))
	if err != nil {
		return nil, err
	}
	return EncodeSubnet(subnet), nil
}

func (t *TypeOfSubnet) ID() int {
	return IdNet
}

func (t *TypeOfSubnet) String() string {
	return "subnet"
}

func (t *TypeOfSubnet) StringOf(zv zcode.Bytes, _ OutFmt, _ bool) string {
	s, err := DecodeSubnet(zv)
	if err != nil {
		return badZng(err, t, zv)
	}
	ipnet := net.IPNet(*s)
	return ipnet.String()
}

func (t *TypeOfSubnet) Marshal(zv zcode.Bytes) (interface{}, error) {
	s, err := DecodeSubnet(zv)
	if err != nil {
		return nil, err
	}
	return (*s).String(), nil
}

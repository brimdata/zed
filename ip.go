package zed

import (
	"net"

	"github.com/brimdata/zed/zcode"
)

type TypeOfIP struct{}

func NewIP(a net.IP) *Value {
	return &Value{TypeIP, EncodeIP(a)}
}

func AppendIP(zb zcode.Bytes, a net.IP) zcode.Bytes {
	ip := a.To4()
	if ip == nil {
		ip = net.IP(a)
	}
	return append(zb, ip...)
}

func EncodeIP(a net.IP) zcode.Bytes {
	return AppendIP(nil, a)
}

func DecodeIP(zv zcode.Bytes) net.IP {
	if zv == nil {
		return nil
	}
	switch len(zv) {
	case 4, 16:
		return net.IP(zv)
	}
	panic("failure trying to decode IP address that is not 4 or 16 bytes long")
}

func (t *TypeOfIP) ID() int {
	return IDIP
}

func (t *TypeOfIP) String() string {
	return "ip"
}

func (t *TypeOfIP) Marshal(zv zcode.Bytes) interface{} {
	return DecodeIP(zv).String()
}

func (t *TypeOfIP) Format(zb zcode.Bytes) string {
	return DecodeIP(zb).String()
}

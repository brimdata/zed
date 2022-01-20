package zed

import (
	"fmt"

	"github.com/brimdata/zed/zcode"
	"inet.af/netaddr"
)

type TypeOfIP struct{}

func NewIP(a netaddr.IP) *Value {
	return &Value{TypeIP, EncodeIP(a)}
}

func AppendIP(zb zcode.Bytes, a netaddr.IP) zcode.Bytes {
	if a.Is4() {
		ip := a.As4()
		return append(zb, ip[:]...)
	}
	ip := a.As16()
	return append(zb, ip[:]...)
}

func EncodeIP(a netaddr.IP) zcode.Bytes {
	return AppendIP(nil, a)
}

func DecodeIP(zv zcode.Bytes) netaddr.IP {
	var ip netaddr.IP
	if err := ip.UnmarshalBinary(zv); err != nil {
		panic(fmt.Errorf("failure trying to decode IP address: %w", err))
	}
	return ip
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

package zng

import (
	"errors"
	"net"

	"github.com/brimsec/zq/zcode"
)

type TypeOfAddr struct{}

func NewAddr(a net.IP) Value {
	return Value{TypeAddr, EncodeAddr(a)}
}

func EncodeAddr(a net.IP) zcode.Bytes {
	ip := a.To4()
	if ip == nil {
		ip = net.IP(a)
	}
	return zcode.Bytes(ip)
}

func DecodeAddr(zv zcode.Bytes) (net.IP, error) {
	if zv == nil {
		return nil, ErrUnset
	}
	switch len(zv) {
	case 4, 16:
		return net.IP(zv), nil
	}
	return nil, errors.New("failure trying to decode IP address that is not 4 or 16 bytes long")
}

func (t *TypeOfAddr) Parse(in []byte) (zcode.Bytes, error) {
	ip, err := UnsafeParseAddr(in)
	if err != nil {
		return nil, err
	}
	return EncodeAddr(ip), nil
}

func (t *TypeOfAddr) ID() int {
	return IdIP
}

func (t *TypeOfAddr) String() string {
	return "addr"
}

func (t *TypeOfAddr) StringOf(zv zcode.Bytes, _ OutFmt, _ bool) string {
	ip, err := DecodeAddr(zv)
	if err != nil {
		return badZng(err, t, zv)
	}
	return ip.String()
}

func (t *TypeOfAddr) Marshal(zv zcode.Bytes) (interface{}, error) {
	ip, err := DecodeAddr(zv)
	if err != nil {
		return nil, err
	}
	return ip.String(), nil
}

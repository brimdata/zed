package zng

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"

	"github.com/mccanne/zq/zcode"
)

type TypeOfAddr struct{}

var compareAddr = map[string]func(net.IP, net.IP) bool{
	"eql":  func(a, b net.IP) bool { return a.Equal(b) },
	"neql": func(a, b net.IP) bool { return !a.Equal(b) },
	"gt":   func(a, b net.IP) bool { return bytes.Compare(a, b) > 0 },
	"gte":  func(a, b net.IP) bool { return bytes.Compare(a, b) >= 0 },
	"lt":   func(a, b net.IP) bool { return bytes.Compare(a, b) < 0 },
	"lte":  func(a, b net.IP) bool { return bytes.Compare(a, b) <= 0 },
}

func (t *TypeOfAddr) String() string {
	return "addr"
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

func (t *TypeOfAddr) New(zv zcode.Bytes) (Value, error) {
	if zv == nil {
		return &Unset{}, nil
	}
	ip, err := DecodeAddr(zv)
	if err != nil {
		return nil, err
	}
	return NewAddr(ip), nil
}

type Addr net.IP

func NewAddr(a net.IP) *Addr {
	p := Addr(a)
	return &p
}

func (a Addr) String() string {
	return (net.IP)(a).String()
}

func (a Addr) Type() Type {
	return TypeAddr
}

func (a Addr) Encode(dst zcode.Bytes) zcode.Bytes {
	return zcode.AppendValue(dst, EncodeAddr(net.IP(a)))
}

// Comparison returns a Predicate that compares typed byte slices that must
// be TypeAddr with the value's address using a comparison based on op.
// Only equality operands are allowed.
func (a Addr) Comparison(op string) (Predicate, error) {
	compare, ok := compareAddr[op]
	if !ok {
		return nil, fmt.Errorf("unknown addr comparator: %s", op)
	}
	pattern := net.IP(a)
	return func(e TypedEncoding) bool {
		if e.Type != TypeAddr {
			return false
		}
		ip, err := DecodeAddr(e.Body)
		if err != nil {
			return false
		}
		return compare(ip, pattern)
	}, nil
}

func (a Addr) Coerce(typ Type) Value {
	_, ok := typ.(*TypeOfAddr)
	if ok {
		return a
	}
	return nil
}

func (a Addr) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

func (a Addr) Elements() ([]Value, bool) { return nil, false }

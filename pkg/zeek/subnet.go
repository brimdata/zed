package zeek

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net"

	"github.com/mccanne/zq/pkg/zval"
)

type TypeOfSubnet struct{}

// a better way to do this would be to compare IP's and mask's but
// go doesn't provide an easy way to compare masks so we do this
// hacky thing and compare strings
var compareSubnet = map[string]func(*net.IPNet, *net.IPNet) bool{
	"eql":  func(a, b *net.IPNet) bool { return bytes.Equal(a.IP, b.IP) },
	"neql": func(a, b *net.IPNet) bool { return bytes.Equal(a.IP, b.IP) },
	"lt":   func(a, b *net.IPNet) bool { return bytes.Compare(a.IP, b.IP) < 0 },
	"lte":  func(a, b *net.IPNet) bool { return bytes.Compare(a.IP, b.IP) <= 0 },
	"gt":   func(a, b *net.IPNet) bool { return bytes.Compare(a.IP, b.IP) > 0 },
	"gte":  func(a, b *net.IPNet) bool { return bytes.Compare(a.IP, b.IP) >= 0 },
}

var matchSubnet = map[string]func(net.IP, *net.IPNet) bool{
	"eql": func(a net.IP, b *net.IPNet) bool {
		return b.IP.Equal(a.Mask(b.Mask))
	},
	"neql": func(a net.IP, b *net.IPNet) bool {
		return !b.IP.Equal(a.Mask(b.Mask))
	},
	"lt": func(a net.IP, b *net.IPNet) bool {
		net := a.Mask(b.Mask)
		return bytes.Compare(net, b.IP) < 0
	},
	"lte": func(a net.IP, b *net.IPNet) bool {
		net := a.Mask(b.Mask)
		return bytes.Compare(net, b.IP) <= 0
	},
	"gt": func(a net.IP, b *net.IPNet) bool {
		net := a.Mask(b.Mask)
		return bytes.Compare(net, b.IP) > 0
	},
	"gte": func(a net.IP, b *net.IPNet) bool {
		net := a.Mask(b.Mask)
		return bytes.Compare(net, b.IP) >= 0
	},
}

func (t *TypeOfSubnet) String() string {
	return "subnet"
}

func EncodeSubnet(subnet *net.IPNet) zval.Encoding {
	var b [32]byte
	ip := subnet.IP.To4()
	if ip != nil {
		copy(b[:], ip)
		// XXX not sure this works
		copy(b[4:], subnet.Mask)
		return b[:8]
	}
	copy(b[:], ip)
	copy(b[16:], subnet.Mask)
	return b[:]
}

func DecodeSubnet(zv zval.Encoding) (*net.IPNet, error) {
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

func (t *TypeOfSubnet) Parse(in []byte) (zval.Encoding, error) {
	_, subnet, err := net.ParseCIDR(string(in))
	if err != nil {
		return nil, err
	}
	return EncodeSubnet(subnet), nil
}

func (t *TypeOfSubnet) New(zv zval.Encoding) (Value, error) {
	if zv == nil {
		return &Unset{}, nil
	}
	subnet, err := DecodeSubnet(zv)
	if err != nil {
		return nil, err
	}
	return NewSubnet(subnet), nil
}

type Subnet net.IPNet

func NewSubnet(s *net.IPNet) *Subnet {
	v := Subnet(*s)
	return &v
}

func (s *Subnet) String() string {
	return s.String()
}

func (s *Subnet) Encode(dst zval.Encoding) zval.Encoding {
	zv := EncodeSubnet((*net.IPNet)(s))
	return zval.AppendValue(dst, zv)
}

func (s *Subnet) Type() Type {
	return TypeSubnet
}

// Comparison returns a Predicate that compares typed byte slices that must
// be an addr or a subnet to the value's subnet value using a comparison
// based on op.  Onluy equalty and inequality are permitted.  If the typed
// byte slice is a subnet, then the comparison is based on strict equality.
// If the typed byte slice is an addr, then the comparison is performed by
// doing a CIDR match on the address with the subnet.
func (s *Subnet) Comparison(op string) (Predicate, error) {
	CompareSubnet, ok1 := compareSubnet[op]
	MatchSubnet, ok2 := matchSubnet[op]
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("unknown subnet comparator: %s", op)
	}
	pattern := (*net.IPNet)(s)
	return func(e TypedEncoding) bool {
		val := e.Body
		switch e.Type.(type) {
		case *TypeOfAddr:
			ip, err := DecodeAddr(val)
			if err == nil {
				return MatchSubnet(ip, pattern)
			}
		case *TypeOfSubnet:
			subnet, err := DecodeSubnet(val)
			if err == nil {
				return CompareSubnet(subnet, pattern)
			}
		}
		return false
	}, nil
}

func (s *Subnet) Coerce(typ Type) Value {
	_, ok := typ.(*TypeOfSubnet)
	if ok {
		return s
	}
	return nil
}

func (s *Subnet) MarshalJSON() ([]byte, error) {
	return json.Marshal((*net.IPNet)(s))
}

func (s *Subnet) Elements() ([]Value, bool) { return nil, false }

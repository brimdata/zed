package zeek

import (
	"bytes"
	"encoding/json"
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

func (t *TypeOfSubnet) Parse(value []byte) (*net.IPNet, error) {
	if value == nil {
		return nil, ErrUnset
	}
	_, subnet, err := net.ParseCIDR(ustring(value))
	return subnet, err
}

func (t *TypeOfSubnet) Format(value []byte) (interface{}, error) {
	return t.Parse(value)
}
func (t *TypeOfSubnet) New(value []byte) (Value, error) {
	if value == nil {
		return &Unset{}, nil
	}
	subnet, err := t.Parse(value)
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
	v := []byte(s.String())
	return zval.AppendValue(dst, v)
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
			ip, err := TypeAddr.Parse(val)
			if err == nil {
				return MatchSubnet(ip, pattern)
			}
		case *TypeOfSubnet:
			subnet, err := TypeSubnet.Parse(val)
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

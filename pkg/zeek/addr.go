package zeek

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"

	"github.com/mccanne/zq/pkg/zval"
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

func (t *TypeOfAddr) Parse(value []byte) (net.IP, error) {
	if value == nil {
		return nil, ErrUnset
	}
	return UnsafeParseAddr(value)
}

func (t *TypeOfAddr) Format(value []byte) (interface{}, error) {
	return t.Parse(value)
}

func (t *TypeOfAddr) New(value []byte) (Value, error) {
	if value == nil {
		return &Unset{}, nil
	}
	ip, err := t.Parse(value)
	if err != nil {
		return nil, err
	}
	return &Addr{Native: ip}, nil
}

type Addr struct {
	Native net.IP
}

func (a *Addr) String() string {
	return a.Native.String()
}

func (a *Addr) Type() Type {
	return TypeAddr
}

func (a *Addr) Encode(dst zval.Encoding) zval.Encoding {
	b := []byte(a.Native.String())
	return zval.AppendValue(dst, b)
}

// Comparison returns a Predicate that compares typed byte slices that must
// be TypeAddr with the value's address using a comparison based on op.
// Only equality operands are allowed.
func (a *Addr) Comparison(op string) (Predicate, error) {
	compare, ok := compareAddr[op]
	if !ok {
		return nil, fmt.Errorf("unknown addr comparator: %s", op)
	}
	pattern := a.Native
	return func(e TypedEncoding) bool {
		typeAddr, ok := e.Type.(*TypeOfAddr)
		if !ok {
			return false
		}
		ip, err := typeAddr.Parse(e.Body)
		if err != nil {
			return false
		}
		return compare(ip, pattern)
	}, nil
}

func (a *Addr) Coerce(typ Type) Value {
	_, ok := typ.(*TypeOfAddr)
	if ok {
		return a
	}
	return nil
}

func (a *Addr) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.Native)
}

func (a *Addr) Elements() ([]Value, bool) { return nil, false }

package vector

import (
	"net/netip"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type IP struct {
	mem
	Typ    zed.Type
	Values []netip.Addr
}

var _ Any = (*IP)(nil)

func NewIP(typ zed.Type, values []netip.Addr) *IP {
	return &IP{Typ: typ, Values: values}
}

func (i *IP) Type() zed.Type {
	return i.Typ
}

func (i *IP) NewBuilder() Builder {
	var off int
	return func(b *zcode.Builder) bool {
		if off >= len(i.Values) {
			return false
		}
		b.Append(zed.EncodeIP(i.Values[off]))
		off++
		return true
	}
}

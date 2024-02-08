package vector

import (
	"net/netip"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type IP struct {
	Values []netip.Addr
	Nulls  *Bool
}

var _ Any = (*IP)(nil)

func NewIP(values []netip.Addr, nulls *Bool) *IP {
	return &IP{Values: values, Nulls: nulls}
}

func (i *IP) Type() zed.Type {
	return zed.TypeIP
}

func (i *IP) Len() uint32 {
	return uint32(len(i.Values))
}

func (i *IP) Serialize(b *zcode.Builder, slot uint32) {
	if i.Nulls != nil && i.Nulls.Value(slot) {
		b.Append(nil)
	} else {
		b.Append(zed.EncodeIP(i.Values[slot]))
	}
}

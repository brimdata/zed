package vector

import (
	"net/netip"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Net struct {
	Values []netip.Prefix
	Nulls  *Bool
}

var _ Any = (*Net)(nil)

func NewNet(values []netip.Prefix, nulls *Bool) *Net {
	return &Net{Values: values, Nulls: nulls}
}

func (n *Net) Type() zed.Type {
	return zed.TypeNet
}

func (n *Net) Len() uint32 {
	return uint32(len(n.Values))
}

func (n *Net) Serialize(b *zcode.Builder, slot uint32) {
	if n.Nulls != nil && n.Nulls.Value(slot) {
		b.Append(nil)
	} else {
		b.Append(zed.EncodeNet(n.Values[slot]))
	}
}

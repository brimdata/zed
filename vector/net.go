package vector

import (
	"net/netip"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Net struct {
	mem
	Typ    zed.Type
	Values []netip.Prefix
}

var _ Any = (*Net)(nil)

func NewNet(typ zed.Type, values []netip.Prefix) *Net {
	return &Net{Typ: typ, Values: values}
}

func (n *Net) Type() zed.Type {
	return n.Typ
}

func (n *Net) NewBuilder() Builder {
	var off int
	return func(b *zcode.Builder) bool {
		if off >= len(n.Values) {
			return false
		}
		b.Append(zed.EncodeNet(n.Values[off]))
		off++
		return true
	}
}

func (n *Net) Key(b []byte, slot int) []byte {
	return zed.AppendNet(b, n.Values[slot])
}

func (n *Net) Length() int {
	return len(n.Values)
}

func (n *Net) Serialize(slot int) *zed.Value {
	return zed.NewValue(n.Typ, zed.EncodeNet(n.Values[slot]))
}

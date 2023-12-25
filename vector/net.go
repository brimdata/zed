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

type DictNet struct {
	Tags   []byte
	Values []netip.Prefix
	Counts []uint32
	Nulls  *Bool
}

var _ Any = (*DictNet)(nil)

func NewDictNet(tags []byte, values []netip.Prefix, counts []uint32, nulls *Bool) *DictNet {
	return &DictNet{Tags: tags, Values: values, Counts: counts, Nulls: nulls}
}

func (d *DictNet) Type() zed.Type {
	return zed.TypeNet
}

func (d *DictNet) Len() uint32 {
	return uint32(len(d.Tags))
}

func (d *DictNet) Value(slot uint32) netip.Prefix {
	return d.Values[d.Tags[slot]]
}

func (d *DictNet) Serialize(b *zcode.Builder, slot uint32) {
	if d.Nulls != nil && d.Nulls.Value(slot) {
		b.Append(nil)
	} else {
		b.Append(zed.EncodeNet(d.Value(slot)))
	}
}

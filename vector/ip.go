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

type DictIP struct {
	Tags   []byte
	Values []netip.Addr
	Counts []uint32
	Nulls  *Bool
}

var _ Any = (*DictIP)(nil)

func NewDictIP(tags []byte, values []netip.Addr, counts []uint32, nulls *Bool) *DictIP {
	return &DictIP{Tags: tags, Values: values, Counts: counts, Nulls: nulls}
}

func (d *DictIP) Type() zed.Type {
	return zed.TypeIP
}

func (d *DictIP) Len() uint32 {
	return uint32(len(d.Tags))
}

func (d *DictIP) Value(slot uint32) netip.Addr {
	return d.Values[d.Tags[slot]]
}

func (d *DictIP) Serialize(b *zcode.Builder, slot uint32) {
	if d.Nulls != nil && d.Nulls.Value(slot) {
		b.Append(nil)
	} else {
		b.Append(zed.EncodeIP(d.Value(slot)))
	}
}

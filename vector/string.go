package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type String struct {
	Offsets []uint32
	Bytes   []byte
	Nulls   *Bool
}

var _ Any = (*String)(nil)

func NewString(offsets []uint32, bytes []byte, nulls *Bool) *String {
	return &String{Offsets: offsets, Bytes: bytes, Nulls: nulls}
}

func (s *String) Type() zed.Type {
	return zed.TypeString
}

func (s *String) Len() uint32 {
	return uint32(len(s.Offsets) - 1)
}

func (s *String) Value(slot uint32) string {
	return string(s.Bytes[s.Offsets[slot]:s.Offsets[slot+1]])
}

func (s *String) Serialize(b *zcode.Builder, slot uint32) {
	if s.Nulls != nil && s.Nulls.Value(slot) {
		b.Append(nil)
	} else {
		b.Append(zed.EncodeString(s.Value(slot)))
	}
}

type DictString struct {
	Tags   []byte
	Offs   []uint32
	Bytes  []byte
	Counts []uint32
	Nulls  *Bool
}

var _ Any = (*DictString)(nil)

func NewDictString(tags []byte, offs []uint32, bytes []byte, counts []uint32, nulls *Bool) *DictString {
	return &DictString{Tags: tags, Offs: offs, Bytes: bytes, Counts: counts, Nulls: nulls}
}

func (d *DictString) Type() zed.Type {
	return zed.TypeString
}

func (d *DictString) Len() uint32 {
	return uint32(len(d.Tags))
}

func (d *DictString) Value(slot uint32) string {
	tag := d.Tags[slot]
	return string(d.Bytes[d.Offs[tag]:d.Offs[tag+1]])
}

func (d *DictString) Serialize(b *zcode.Builder, slot uint32) {
	if d.Nulls != nil && d.Nulls.Value(slot) {
		b.Append(nil)
	} else {
		b.Append(zed.EncodeString(d.Value(slot)))
	}
}

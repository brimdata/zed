package vector

import (
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Bool struct {
	len   uint32
	bits  []uint64
	Nulls *Bool
}

var _ Any = (*Bool)(nil)

func NewBool(bits []uint64, len uint32, nulls *Bool) *Bool {
	return &Bool{len: len, bits: bits, Nulls: nulls}
}

func NewBoolEmpty(length uint32, nulls *Bool) *Bool {
	return &Bool{len: length, bits: make([]uint64, (length+63)/64), Nulls: nulls}
}

func (b *Bool) Type() zed.Type {
	return zed.TypeBool
}

func (b *Bool) Value(slot uint32) bool {
	return (b.bits[slot>>6] & (1 << (slot & 0x3f))) != 0
}

func (b *Bool) Set(slot uint32) {
	b.bits[slot>>6] |= (1 << (slot & 0x3f))
}

func (b *Bool) Len() uint32 {
	return b.len
}

func (b *Bool) Serialize(builder *zcode.Builder, slot uint32) {
	if b.Nulls != nil && b.Nulls.Value(slot) {
		builder.Append(nil)
	} else {
		builder.Append(zed.EncodeBool(b.Value(slot)))
	}
}

// helpful to have around for debugging
func (b *Bool) String() string {
	var s strings.Builder
	if b == nil || b.Len() == 0 {
		return "empty"
	}
	for k := uint32(0); k < b.Len(); k++ {
		if b.Value(k) {
			s.WriteByte('1')
		} else {
			s.WriteByte('0')
		}
	}
	return s.String()
}

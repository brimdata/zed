package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Nulls struct {
	mem
	bitmap []byte //XXX change to uint64
	nvals  int
	values Any
}

var _ Any = (*Nulls)(nil)

func NewNulls(slots []uint32, nvals int, values Any) *Nulls {
	var nulls []byte
	if len(slots) > 0 {
		nulls = make([]byte, (nvals+7)/8)
		for _, slot := range slots {
			nulls[slot>>3] |= 1 << (slot & 7)
		}
	}
	return &Nulls{bitmap: nulls, nvals: nvals, values: values}
}

func (n *Nulls) Type() zed.Type {
	return n.values.Type()
}

func (n *Nulls) Values() Any {
	return n.values
}

func (n *Nulls) NewBuilder() Builder {
	valueBuilder := n.values.NewBuilder()
	var off int
	return func(b *zcode.Builder) bool {
		if off >= n.nvals {
			return false
		}
		if n.IsNull(off) {
			b.Append(nil)
		} else if !valueBuilder(b) {
			return false
		}
		off++
		return true
	}
}

func (n *Nulls) IsNull(slot int) bool {
	if int(slot) > n.nvals {
		return false
	}
	off := slot / 8
	pos := slot & 7
	return n.bitmap[off]&(1<<pos) != 0
}

//XXX this here is why this nulls wrapper stuff doesn't work.
//We need to move the nulls into the leaves (and keep them
// where need at intermediate nodes too)

func (n *Nulls) Key(b []byte, slot int) []byte {
	panic("TBD")
}

func (n *Nulls) Length() int {
	panic("TBD")
}

func (n *Nulls) Serialize(slot int) *zed.Value {
	panic("TBD")
}

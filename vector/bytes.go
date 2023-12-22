package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Bytes struct {
	mem
	Typ    zed.Type
	Values [][]byte
}

var _ Any = (*Bytes)(nil)

func NewBytes(typ zed.Type, values [][]byte) *Bytes {
	return &Bytes{Typ: typ, Values: values}
}

func (b *Bytes) Type() zed.Type {
	return b.Typ
}

func (b *Bytes) NewBuilder() Builder {
	var off int
	return func(zb *zcode.Builder) bool {
		if off >= len(b.Values) {
			return false
		}
		zb.Append(zed.EncodeBytes(b.Values[off]))
		off++
		return true
	}
}

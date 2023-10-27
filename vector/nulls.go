package vector

import (
	"github.com/brimdata/zed/zcode"
)

type Nulls struct {
	Len    int
	Mask   []byte
	Values Any
}

var _ Any = (*Nulls)(nil)

func (vector *Nulls) Has(index uint32) bool {
	maskIndex := index >> 3
	if maskIndex >= uint32(vector.Len) {
		return false
	}
	bitIndex := index & 0b111
	return (vector.Mask[maskIndex] & (1 << bitIndex)) != 0
}

func (vector *Nulls) newBuilder() builder {
	var index int
	valueBuilder := vector.Values.newBuilder()
	return func(builder *zcode.Builder) {
		if vector.Has(uint32(index)) {
			valueBuilder(builder)
		} else {
			builder.Append(nil)
		}
		index += 1
	}
}

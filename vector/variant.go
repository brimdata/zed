package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

// Variant is an ordered sequence of values taken from one or more
// hetereogenously-typed vectors.
type Variant struct {
	Tags   []uint32
	Values []Any
	TagMap *TagMap
}

var _ Any = (*Variant)(nil)

func NewVariant(tags []uint32, values []Any) *Variant {
	return &Variant{Tags: tags, Values: values, TagMap: NewTagMap(tags, values)}
}

func (v *Variant) Type() zed.Type {
	panic("can't call Type() on a vector.Variant")
}

func (v *Variant) TypeOf(slot uint32) zed.Type {
	vals := v.Values[v.Tags[slot]]
	if v2, ok := vals.(*Variant); ok {
		return v2.TypeOf(v.TagMap.Forward[slot])
	}
	return vals.Type()
}

func (v *Variant) Len() uint32 {
	if v.Tags != nil {
		return uint32(len(v.Tags))
	}
	var length uint32
	for _, val := range v.Values {
		length += val.Len()
	}
	return length
}

func (v *Variant) Serialize(b *zcode.Builder, slot uint32) {
	v.Values[v.Tags[slot]].Serialize(b, v.TagMap.Forward[slot])
}

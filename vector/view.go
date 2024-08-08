package vector

import (
	"github.com/brimdata/zed/zcode"
)

type View struct {
	Any
	Index []uint32
}

var _ Any = (*View)(nil)

func NewView(index []uint32, val Any) Any {
	switch val := val.(type) {
	case *Const:
		return NewConst(val.arena, val.Value(), uint32(len(index)), nil)
	case *Dict:
		index2 := make([]uint32, len(index))
		for k, idx := range index {
			index2[k] = uint32(val.Index[idx])
		}
		return &View{val.Any, index2}
	case *Union:
		tags, values := viewForUnionOrVariant(index, val.Tags, val.TagMap.Forward, val.Values)
		return NewUnion(val.Typ, tags, values, nil)
	case *Variant:
		return NewVariant(viewForUnionOrVariant(index, val.Tags, val.TagMap.Forward, val.Values))
	case *View:
		index2 := make([]uint32, len(index))
		for k, idx := range index {
			index2[k] = uint32(val.Index[idx])
		}
		return &View{val.Any, index2}
	}
	return &View{val, index}
}

func viewForUnionOrVariant(index, tags, forward []uint32, values []Any) ([]uint32, []Any) {
	indexes := make([][]uint32, len(values))
	resultTags := make([]uint32, len(index))
	for k, index := range index {
		tag := tags[index]
		indexes[tag] = append(indexes[tag], forward[index])
		resultTags[k] = tag
	}
	results := make([]Any, len(values))
	for k := range results {
		results[k] = NewView(indexes[k], values[k])
	}
	return resultTags, results
}

func (v *View) Len() uint32 {
	return uint32(len(v.Index))
}

func (v *View) Serialize(b *zcode.Builder, slot uint32) {
	v.Any.Serialize(b, v.Index[slot])
}

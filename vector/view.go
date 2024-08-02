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
	case *View:
		index2 := make([]uint32, len(index))
		for k, idx := range index {
			index2[k] = uint32(val.Index[idx])
		}
		return &View{val.Any, index2}
	}
	return &View{val, index}
}

func (v *View) Len() uint32 {
	return uint32(len(v.Index))
}

func (v *View) Serialize(b *zcode.Builder, slot uint32) {
	v.Any.Serialize(b, v.Index[slot])
}

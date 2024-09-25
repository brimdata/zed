package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Any interface {
	Type() zed.Type
	Len() uint32
	Serialize(*zcode.Builder, uint32)
}

type Promotable interface {
	Any
	Promote(zed.Type) Promotable
}

type Puller interface {
	Pull(done bool) (Any, error)
}

type Builder func(*zcode.Builder) bool

func Combine(vec Any, index []uint32, add Any) Any {
	var vecs []Any
	tags := make([]uint32, int(vec.Len())+len(index))
	if d, ok := vec.(*Dynamic); ok {
		vecs = d.Values
		varTags := d.Tags
		n := uint32(len(vecs))
		for i := uint32(0); i < uint32(len(tags)); i++ {
			if len(index) > 0 && i == index[0] {
				tags[i] = n
				index = index[1:]
			} else {
				tags[i] = varTags[0]
				varTags = varTags[1:]
			}
		}
	} else {
		vecs = []Any{vec}
		for _, k := range index {
			tags[k] = 1
		}
	}
	return NewDynamic(tags, append(vecs, add))
}

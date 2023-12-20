package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Union struct {
	mem
	Typ    *zed.TypeUnion
	Tags   []int32
	Values []Any
}

var _ Any = (*Union)(nil)

func NewUnion(typ *zed.TypeUnion) *Union {
	return &Union{Typ: typ, Values: make([]Any, len(typ.Types))}
}

func (u *Union) Type() zed.Type {
	return u.Typ
}

func (u *Union) NewBuilder() Builder {
	var valueBuilders []Builder
	for _, v := range u.Values {
		valueBuilders = append(valueBuilders, v.NewBuilder())
	}
	var off int
	return func(b *zcode.Builder) bool {
		if off >= len(u.Tags) {
			return false
		}
		tag := u.Tags[off]
		b.BeginContainer()
		b.Append(zed.EncodeInt(int64(tag)))
		if !valueBuilders[tag](b) {
			return false
		}
		b.EndContainer()
		off++
		return true
	}
}

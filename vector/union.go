package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Union struct {
	*Variant
	Typ   *zed.TypeUnion
	Nulls *Bool
}

var _ Any = (*Union)(nil)

func NewUnion(typ *zed.TypeUnion, tags []uint32, vals []Any, nulls *Bool) *Union {
	return &Union{NewVariant(tags, vals), typ, nulls}
}

func (u *Union) Type() zed.Type {
	return u.Typ
}

func (u *Union) Serialize(b *zcode.Builder, slot uint32) {
	b.BeginContainer()
	b.Append(zed.EncodeInt(int64(u.Tags[slot])))
	u.Variant.Serialize(b, slot)
	b.EndContainer()
}

func Deunion(vec Any) Any {
	if union, ok := vec.(*Union); ok {
		// XXX if the Union has Nulls this will be broken.
		return union.Variant
	}
	return vec
}

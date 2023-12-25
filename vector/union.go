package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Union struct {
	Typ    *zed.TypeUnion
	Tags   []uint32
	TagMap TagMap
	Values []Any
	Nulls  *Bool
}

var _ Any = (*Union)(nil)

func NewUnion(typ *zed.TypeUnion, tags []uint32, vals []Any, nulls *Bool) *Union {
	return &Union{
		Typ:    typ,
		Tags:   tags,
		TagMap: *NewTagMap(tags, vals),
		Values: vals,
	}
}

func (u *Union) Type() zed.Type {
	return u.Typ
}

func (u *Union) Len() uint32 {
	return uint32(len(u.Tags))
}

func (u *Union) Serialize(b *zcode.Builder, slot uint32) {
	tag := u.Tags[slot]
	b.BeginContainer()
	b.Append(zed.EncodeInt(int64(tag)))
	u.Values[tag].Serialize(b, u.TagMap.Forward[slot])
	b.EndContainer()
}

func (u *Union) Copy(vals []Any) *Union {
	return &Union{
		Typ:    u.Typ,
		Tags:   u.Tags,
		TagMap: u.TagMap,
		Values: vals,
	}
}

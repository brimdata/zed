package vector

import (
	"fmt"

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
	return &Union{typ, tags, *NewTagMap(tags, vals), vals, nulls}
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
	u2 := *u
	u2.Values = vals
	return &u2
}

//XXX should stich/unstitch be methods on tagmap?

//XXX handle union where there are no values for one of the types

// Unstitch returns a set of views one for each type in the union
// such that the input vector v is the same length as u and each
// new view of v is congruent with the corresponding union member vector.
func (u *Union) Unstitch(v Any) []*View {
	// We can simply use the reverse tagmap to create the views into v.
	if v.Len() != u.Len() {
		panic(fmt.Sprintf("vector.Union.Unpack mismatched vector sizes: %d vs %d", u.Len(), v.Len()))
	}
	n := len(u.Values)
	views := make([]*View, n)
	for k := 0; k < n; k++ {
		views[k] = NewView(u.TagMap.Reverse[k], v)
	}
	return views
}

// XXX len(v) must be the same as len(u.Values) and len(v[k]) = len(u.Values[tag])
//XXX resolve this with the idea of the variant sequence... we can have a varseq
// as a result of an expr without there being a union type... and we might as well 
// allow there to be multiple of the same type in the stitch/varseq to simplify 
// things and preserve the reverse tagmap (the stitch tags).
func (u *Union) Stitch(zctx *zed.Context, inputs []Any) Any {
	n := len(u.Values)
	views := make([]*View, n)
	types := make(map[zed.Type]struct{})
	for _, v := range inputs {
		types[v.Type()] = struct{}{}
	}
	if len(types)
	for k := 0; k < n; k++ {
		views[k] = NewView(u.TagMap.Reverse[k], v)
	}
	return views
}

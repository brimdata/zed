package zed

import (
	"github.com/brimdata/zed/zcode"
)

type TypeUnion struct {
	id    int
	Types []Type
}

func NewTypeUnion(id int, types []Type) *TypeUnion {
	return &TypeUnion{id, types}
}

func (t *TypeUnion) ID() int {
	return t.id
}

// Type returns the type corresponding to selector.
func (t *TypeUnion) Type(selector int) (Type, error) {
	if selector < 0 || selector >= len(t.Types) {
		return nil, ErrUnionSelector
	}
	return t.Types[selector], nil
}

// SplitZNG takes a zng encoding of a value of the receiver's union type and
// returns the concrete type of the value, its selector, and the value encoding.
func (t *TypeUnion) SplitZNG(zv zcode.Bytes) (Type, int64, zcode.Bytes, error) {
	it := zv.Iter()
	selector := DecodeInt(it.Next())
	inner, err := t.Type(int(selector))
	if err != nil {
		return nil, -1, nil, err
	}
	return inner, selector, it.Next(), nil
}

func (t *TypeUnion) Kind() string {
	return "union"
}

// BuildUnion appends to b a union described by selector and val.
func BuildUnion(b *zcode.Builder, selector int, val zcode.Bytes) {
	if val == nil {
		b.Append(nil)
		return
	}
	b.BeginContainer()
	b.Append(EncodeInt(int64(selector)))
	b.Append(val)
	b.EndContainer()
}

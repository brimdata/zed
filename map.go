package zed

import (
	"bytes"
	"sort"

	"github.com/brimdata/zed/zcode"
)

type TypeMap struct {
	id      int
	KeyType Type
	ValType Type
}

func NewTypeMap(id int, keyType, valType Type) *TypeMap {
	return &TypeMap{id, keyType, valType}
}

func (t *TypeMap) ID() int {
	return t.id
}

func (t *TypeMap) Kind() Kind {
	return MapKind
}

func (t *TypeMap) Decode(zv zcode.Bytes) (Value, Value, error) {
	if zv == nil {
		return Value{}, Value{}, nil
	}
	it := zv.Iter()
	return Value{t.KeyType, it.Next()}, Value{t.ValType, it.Next()}, nil
}

type keyval struct {
	key zcode.Bytes
	val zcode.Bytes
}

// NormalizeMap interprets zv as a map body and returns an equivalent map body
// that is normalized according to the ZNG specification (i.e., the tag-counted
// value of each entry's key is lexicographically greater than that of the
// preceding entry).
func NormalizeMap(zv zcode.Bytes) zcode.Bytes {
	elements := make([]keyval, 0, 8)
	for it := zv.Iter(); !it.Done(); {
		key := it.NextTagAndBody()
		val := it.NextTagAndBody()
		elements = append(elements, keyval{key, val})
	}
	if len(elements) < 2 {
		return zv
	}
	sort.Slice(elements, func(i, j int) bool {
		return bytes.Compare(elements[i].key, elements[j].key) == -1
	})
	norm := make(zcode.Bytes, 0, len(zv))
	norm = append(norm, elements[0].key...)
	norm = append(norm, elements[0].val...)
	for i := 1; i < len(elements); i++ {
		// Skip duplicates.
		if !bytes.Equal(elements[i].key, elements[i-1].key) {
			norm = append(norm, elements[i].key...)
			norm = append(norm, elements[i].val...)
		}
	}
	return norm
}

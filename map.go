package zed

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

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

func (t *TypeMap) String() string {
	return fmt.Sprintf("|{%s:%s|}", t.KeyType, t.ValType)
}

func (t *TypeMap) Decode(zv zcode.Bytes) (Value, Value, error) {
	if zv == nil {
		return Value{}, Value{}, nil
	}
	it := zv.Iter()
	key := it.Next()
	val := it.Next()
	return Value{t.KeyType, key}, Value{t.ValType, val}, nil
}

func (t *TypeMap) Marshal(zv zcode.Bytes) interface{} {
	// start out with zero-length container so we get "[]" instead of nil
	vals := []*Value{}
	it := zv.Iter()
	for !it.Done() {
		vals = append(vals, &Value{t.KeyType, it.Next()})
		vals = append(vals, &Value{t.ValType, it.Next()})
	}
	return vals
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

func (t *TypeMap) Format(zv zcode.Bytes) string {
	var b strings.Builder
	it := zv.Iter()
	b.WriteString("|{")
	sep := ""
	for !it.Done() {
		val := it.Next()
		b.WriteString(sep)
		b.WriteByte('{')
		b.WriteString(t.KeyType.Format(val))
		b.WriteByte(',')
		val = it.Next()
		b.WriteString(t.ValType.Format(val))
		b.WriteByte('}')
		b.WriteString(sep)
		sep = ","
	}
	b.WriteString("}|")
	return b.String()
}

package zed

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/brimdata/zed/zcode"
)

type TypeSet struct {
	id   int
	Type Type
}

func NewTypeSet(id int, typ Type) *TypeSet {
	return &TypeSet{id, typ}
}

func (t *TypeSet) ID() int {
	return t.id
}

func (t *TypeSet) String() string {
	return fmt.Sprintf("|[%s]|", t.Type)
}

func (t *TypeSet) Marshal(zv zcode.Bytes) interface{} {
	// start out with zero-length container so we get "[]" instead of nil
	vals := []*Value{}
	it := zv.Iter()
	for !it.Done() {
		vals = append(vals, &Value{t.Type, it.Next()})
	}
	return vals
}

func (t *TypeSet) Format(zv zcode.Bytes) string {
	var b strings.Builder
	b.WriteString("|[")
	sep := ""
	it := zv.Iter()
	for !it.Done() {
		b.WriteString(sep)
		if val := it.Next(); val == nil {
			b.WriteString("null")
		} else {
			b.WriteString(t.Type.Format(val))
		}
		sep = ","
	}
	b.WriteString("]|")
	return b.String()
}

// NormalizeSet interprets zv as a set body and returns an equivalent set body
// that is normalized according to the ZNG specification (i.e., each element's
// tag-counted value is lexicographically greater than that of the preceding
// element).
func NormalizeSet(zv zcode.Bytes) zcode.Bytes {
	elements := make([]zcode.Bytes, 0, 8)
	for it := zv.Iter(); !it.Done(); {
		elements = append(elements, it.NextTagAndBody())
	}
	if len(elements) < 2 {
		return zv
	}
	sort.Slice(elements, func(i, j int) bool {
		return bytes.Compare(elements[i], elements[j]) == -1
	})
	norm := make(zcode.Bytes, 0, len(zv))
	norm = append(norm, elements[0]...)
	for i := 1; i < len(elements); i++ {
		// Skip duplicates.
		if !bytes.Equal(elements[i], elements[i-1]) {
			norm = append(norm, elements[i]...)
		}
	}
	return norm
}

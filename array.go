package zed

import (
	"fmt"
	"strings"

	"github.com/brimdata/zed/zcode"
)

type TypeArray struct {
	id   int
	Type Type
}

func NewTypeArray(id int, typ Type) *TypeArray {
	return &TypeArray{id, typ}
}

func (t *TypeArray) ID() int {
	return t.id
}

func (t *TypeArray) String() string {
	return fmt.Sprintf("[%s]", t.Type)
}

func (t *TypeArray) Format(zv zcode.Bytes) string {
	var b strings.Builder
	sep := ""
	b.WriteByte('[')
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
	b.WriteByte(']')
	return b.String()
}

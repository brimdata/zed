package zng

import (
	"fmt"
	"strings"

	"github.com/brimsec/zq/alpha/zcode"
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

//XXX get rid of this when we implement full ZNG
func (t *TypeArray) SetID(id int) {
	t.id = id
}

func (t *TypeArray) String() string {
	return fmt.Sprintf("array[%s]", t.Type)
}

func (t *TypeArray) Decode(zv zcode.Bytes) ([]Value, error) {
	if zv == nil {
		return nil, ErrUnset
	}
	return parseContainer(t, t.Type, zv)
}

func (t *TypeArray) Parse(in []byte) (zcode.Bytes, error) {
	panic("zeek.TypeArray.Parse shouldn't be called")
}

func (t *TypeArray) StringOf(zv zcode.Bytes, fmt OutFmt, _ bool) string {
	if len(zv) == 0 && (fmt == OutFormatZeek || fmt == OutFormatZeekAscii) {
		return "(empty)"
	}

	var b strings.Builder
	separator := byte(',')
	switch fmt {
	case OutFormatZNG:
		b.WriteByte('[')
		separator = ';'
	case OutFormatDebug:
		b.WriteString("array[")
	}

	first := true
	it := zv.Iter()
	for !it.Done() {
		val, _, err := it.Next()
		if err != nil {
			//XXX
			b.WriteString("ERR")
			break
		}
		if first {
			first = false
		} else {
			b.WriteByte(separator)
		}
		if val == nil {
			b.WriteByte('-')
		} else {
			b.WriteString(t.Type.StringOf(val, fmt, true))
		}
	}

	switch fmt {
	case OutFormatZNG:
		if !first {
			b.WriteByte(';')
		}
		b.WriteByte(']')
	case OutFormatDebug:
		b.WriteByte(']')
	}
	return b.String()
}

func (t *TypeArray) Marshal(zv zcode.Bytes) (interface{}, error) {
	// start out with zero-length container so we get "[]" instead of nil
	vals := make([]Value, 0)
	it := zv.Iter()
	for !it.Done() {
		val, _, err := it.Next()
		if err != nil {
			return nil, err
		}
		vals = append(vals, Value{t.Type, val})
	}
	return vals, nil
}

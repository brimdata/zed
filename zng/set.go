package zng

import (
	"fmt"
	"strings"

	"github.com/mccanne/zq/zcode"
)

type TypeSet struct {
	id        int
	InnerType Type
}

func NewTypeSet(id int, typ Type) *TypeSet {
	return &TypeSet{id, typ}
}

func (t *TypeSet) ID() int {
	return t.id
}

//XXX get rid of this when we implement full ZNG
func (t *TypeSet) SetID(id int) {
	t.id = id
}

func (t *TypeSet) String() string {
	return fmt.Sprintf("set[%s]", t.InnerType)
}
func (t *TypeSet) Decode(zv zcode.Bytes) ([]Value, error) {
	if zv == nil {
		return nil, ErrUnset
	}
	return parseContainer(t, t.InnerType, zv)
}

func (t *TypeSet) Parse(in []byte) (zcode.Bytes, error) {
	panic("zeek.TypeSet.Parse shouldn't be called")
}

func (t *TypeSet) StringOf(zv zcode.Bytes, fmt OutFmt) string {
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
		b.WriteString("set[")
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
		b.WriteString(t.InnerType.StringOf(val, fmt))
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

func (t *TypeSet) Marshal(zv zcode.Bytes) (interface{}, error) {
	// start out with zero-length container so we get "[]" instead of nil
	vals := make([]Value, 0)
	it := zv.Iter()
	for !it.Done() {
		val, _, err := it.Next()
		if err != nil {
			return nil, err
		}
		vals = append(vals, Value{t.InnerType, val})
	}
	return vals, nil
}

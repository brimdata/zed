package zng

import (
	"fmt"
	"strings"

	"github.com/mccanne/zq/zcode"
)

type TypeVector struct {
	id   int
	Type Type
}

func NewTypeVector(id int, typ Type) *TypeVector {
	return &TypeVector{id, typ}
}

func (t *TypeVector) ID() int {
	return t.id
}

//XXX get rid of this when we implement full ZNG
func (t *TypeVector) SetID(id int) {
	t.id = id
}

func (t *TypeVector) String() string {
	return fmt.Sprintf("vector[%s]", t.Type)
}

func (t *TypeVector) Decode(zv zcode.Bytes) ([]Value, error) {
	if zv == nil {
		return nil, ErrUnset
	}
	return parseContainer(t, t.Type, zv)
}

func (t *TypeVector) Parse(in []byte) (zcode.Bytes, error) {
	panic("zeek.TypeVector.Parse shouldn't be called")
}

func (t *TypeVector) StringOf(zv zcode.Bytes, fmt OutFmt) string {
	if len(zv) == 0 && fmt == OUT_FORMAT_ZEEK || fmt == OUT_FORMAT_ZEEK_ASCII {
		return "(empty)"
	}

	var b strings.Builder
	separator := byte(',')
	switch fmt {
	case OUT_FORMAT_ZNG:
		b.WriteByte('[')
		separator = ';'
	case OUT_FORMAT_DEBUG:
		b.WriteString("vector[")
	}

	first := true
	it := zv.Iter()
	for !it.Done() {
		val, container, err := it.Next()
		if container || err != nil {
			//XXX
			b.WriteString("ERR")
			break
		}
		if first {
			first = false
		} else {
			b.WriteByte(separator)
		}
		if len(val) == 0 {
			b.WriteByte('-')
		} else {
			b.WriteString(t.Type.StringOf(val, fmt))
		}
	}

	switch fmt {
	case OUT_FORMAT_ZNG, OUT_FORMAT_DEBUG:
		b.WriteByte(']')
	}
	return b.String()
}

func (t *TypeVector) Marshal(zv zcode.Bytes) (interface{}, error) {
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

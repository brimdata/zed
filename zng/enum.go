package zng

import (
	"errors"
	"strings"

	"github.com/brimdata/zed/zcode"
)

type TypeEnum struct {
	id       int
	Type     Type
	Elements []Element
}

type Element struct {
	Name  string
	Value zcode.Bytes
}

func NewTypeEnum(id int, typ Type, elements []Element) *TypeEnum {
	return &TypeEnum{id, typ, elements}
}

func (t *TypeEnum) ID() int {
	return t.id
}

func (t *TypeEnum) Element(index int) (*Element, error) {
	if index < 0 || index >= len(t.Elements) {
		return nil, ErrEnumIndex
	}
	return &t.Elements[index], nil
}

func (t *TypeEnum) Marshal(zv zcode.Bytes) (interface{}, error) {
	return TypeUint64.Marshal(zv)
}

func (t *TypeEnum) String() string {
	var b strings.Builder
	typ := t.Type
	b.WriteByte('<')
	sep := ""
	for _, e := range t.Elements {
		b.WriteString(sep)
		b.WriteString(QuotedName(e.Name))
		b.WriteByte(':')
		b.WriteString(typ.Format(e.Value))
		sep = ","
	}
	b.WriteByte('>')
	return b.String()
}

func (t *TypeEnum) Format(zv zcode.Bytes) string {
	id, err := DecodeUint(zv)
	if id >= uint64(len(t.Elements)) || err != nil {
		if err == nil {
			err = errors.New("enum index out of range")
		}
		return badZng(err, t, zv)
	}
	return t.Elements[id].Name
}

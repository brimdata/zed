package zng

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"github.com/mccanne/zq/zcode"
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

func (t *TypeUnion) SetID(id int) {
	t.id = id
}

func (t *TypeUnion) TypeIndex(index int) (Type, error) {
	if index < 0 || index+1 > len(t.Types) {
		return nil, ErrUnionIndex
	}
	return t.Types[index], nil

}

func (t *TypeUnion) String() string {
	var ss []string
	for _, typ := range t.Types {
		ss = append(ss, typ.String())
	}
	return fmt.Sprintf("union[%s]", strings.Join(ss, ","))
}

func (t *TypeUnion) Parse(in []byte) (zcode.Bytes, error) {
	c := bytes.Index(in, []byte{':'})
	if c < 0 {
		return nil, ErrBadValue
	}
	index, err := strconv.Atoi(string(in[0:c]))
	if err != nil {
		return nil, err
	}
	typ, err := t.TypeIndex(index)
	if err != nil {
		return nil, err
	}
	out := zcode.AppendUvarint(zcode.Bytes{}, uint64(index))
	val, err := typ.Parse(in[c+1:])
	if err != nil {
		return nil, err
	}
	out = append(out, val...)
	return out, nil
}

func (t *TypeUnion) StringOf(zv zcode.Bytes, ofmt OutFmt, inContainer bool) string {
	index, n := binary.Uvarint(zv)
	innerType, err := t.TypeIndex(int(index))
	if err != nil {
		return "ERR"
	}
	zv = zv[n:]
	s := strconv.FormatUint(index, 10) + ":"
	return s + innerType.StringOf(zv, ofmt, false)
}

func (t *TypeUnion) Marshal(zv zcode.Bytes) (interface{}, error) {
	index, n := binary.Uvarint(zv)
	innerType, err := t.TypeIndex(int(index))
	if err != nil {
		return nil, err
	}
	zv = zv[n:]
	return innerType.Marshal(zv)
}

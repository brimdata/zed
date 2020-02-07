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
	if index < 0 || index >= len(t.Types) {
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
	panic("TypeUnion.Parse shouldn't be called")
}

// SplitBzng takes a bzng encoding of a value of the receiver's union type and
// returns the concrete type of the value, its index into the union
// type, and the value encoding.
func (t *TypeUnion) SplitBzng(zv zcode.Bytes) (Type, int64, zcode.Bytes, error) {
	it := zcode.Iter(zv)
	v, container, err := it.Next()
	if err != nil {
		return nil, -1, nil, err
	}
	if container {
		return nil, -1, nil, ErrBadValue
	}
	index := zcode.DecodeCountedUvarint(v)
	inner, err := t.TypeIndex(int(index))
	if err != nil {
		return nil, -1, nil, err
	}
	v, _, err = it.Next()
	if err != nil {
		return nil, -1, nil, err
	}
	if !it.Done() {
		return nil, -1, nil, ErrBadValue
	}
	return inner, int64(index), v, nil
}

// SplitZng takes a zng encoding of a value of the receiver's type and returns the
// concrete type of the value, its index into the union type, and the value
// encoding.
func (t *TypeUnion) SplitZng(in []byte) (Type, int, []byte, error) {
	c := bytes.Index(in, []byte{':'})
	if c < 0 {
		return nil, -1, nil, ErrBadValue
	}
	index, err := strconv.Atoi(string(in[0:c]))
	if err != nil {
		return nil, -1, nil, err
	}
	typ, err := t.TypeIndex(index)
	if err != nil {
		return nil, -1, nil, err
	}
	return typ, index, in[c+1:], nil
}

func (t *TypeUnion) StringOf(zv zcode.Bytes, ofmt OutFmt, inContainer bool) string {
	index, n := binary.Uvarint(zv)
	if n < 0 {
		// this follows set and record StringOfs. Like there, XXX.
		return "ERR"
	}
	innerType, err := t.TypeIndex(int(index))
	if err != nil {
		return "ERR"
	}
	zv = zv[n:]
	s := strconv.FormatUint(index, 10) + ":"
	return s + innerType.StringOf(zv, ofmt, false)
}

func (t *TypeUnion) Marshal(zv zcode.Bytes) (interface{}, error) {
	inner, _, zv, err := t.SplitBzng(zv)
	if err != nil {
		return nil, err
	}
	return inner.Marshal(zv)
}

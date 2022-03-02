package expr

import (
	"errors"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/expr/coerce"
	"github.com/brimdata/zed/zcode"
)

type Slice struct {
	zctx *zed.Context
	elem Evaluator
	from Evaluator
	to   Evaluator
}

func NewSlice(zctx *zed.Context, elem, from, to Evaluator) *Slice {
	return &Slice{
		zctx: zctx,
		elem: elem,
		from: from,
		to:   to,
	}
}

var ErrSliceIndex = errors.New("slice index is not a number")
var ErrSliceIndexEmpty = errors.New("slice index is empty")

func (s *Slice) Eval(ectx Context, this *zed.Value) *zed.Value {
	elem := s.elem.Eval(ectx, this)
	if elem.IsError() {
		return elem
	}
	var length int
	switch zed.TypeUnder(elem.Type).(type) {
	case *zed.TypeOfBytes, *zed.TypeOfString:
		length = len(elem.Bytes)
	case *zed.TypeArray:
		n, err := elem.ContainerLength()
		if err != nil {
			panic(err)
		}
		length = n
	default:
		return s.zctx.WrapError("sliced value is not array, bytes, or string", elem)
	}
	if elem.IsNull() {
		return elem
	}
	from, err := sliceIndex(ectx, this, s.from, length)
	if err != nil && err != ErrSliceIndexEmpty {
		return s.zctx.NewError(err)
	}
	to, err := sliceIndex(ectx, this, s.to, length)
	if err != nil {
		if err != ErrSliceIndexEmpty {
			return s.zctx.NewError(err)
		}
		to = length
	}
	bytes := elem.Bytes
	if _, ok := zed.TypeUnder(elem.Type).(*zed.TypeArray); ok {
		it := bytes.Iter()
		for k := 0; k < to && !it.Done(); k++ {
			if k == from {
				bytes = zcode.Bytes(it)
			}
			it.Next()
		}
		bytes = bytes[:len(bytes)-len(it)]
	} else {
		bytes = bytes[from:to]
	}
	return ectx.NewValue(elem.Type, bytes)

}

func sliceIndex(ectx Context, this *zed.Value, slot Evaluator, length int) (int, error) {
	if slot == nil {
		//XXX
		return 0, ErrSliceIndexEmpty
	}
	zv := slot.Eval(ectx, this)
	v, ok := coerce.ToInt(*zv)
	if !ok {
		return 0, ErrSliceIndex
	}
	index := int(v)
	if index < 0 {
		index += length
	}
	if index < 0 {
		return 0, nil
	}
	if index > length {
		return length, nil
	}
	return index, nil
}

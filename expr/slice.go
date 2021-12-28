package expr

import (
	"errors"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr/coerce"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

type Slice struct {
	elem Evaluator
	from Evaluator
	to   Evaluator
}

func NewSlice(elem, from, to Evaluator) *Slice {
	return &Slice{
		elem: elem,
		from: from,
		to:   to,
	}
}

var ErrSliceIndex = errors.New("array slice is not a number")
var ErrSliceIndexEmpty = errors.New("array slice is empty")

func sliceIndex(ectx Context, this *zed.Value, slot Evaluator, elem *zed.Value) (int, error) {
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
		n, err := elem.ContainerLength()
		if err != nil {
			return 0, err
		}
		index += n
	}
	return index, nil
}

func (s *Slice) Eval(ectx Context, this *zed.Value) *zed.Value {
	elem := s.elem.Eval(ectx, this)
	if elem.IsError() {
		return elem
	}
	if _, ok := zed.AliasOf(elem.Type).(*zed.TypeArray); !ok {
		// XXX use structured error
		return zed.NewErrorf("sliced value is not an array: %s", zson.MustFormatValue(*elem))
	}
	if elem.IsNull() {
		// If array is null, just return the null array.
		return elem
	}
	from, err := sliceIndex(ectx, this, s.from, elem)
	if err != nil && err != ErrSliceIndexEmpty {
		return zed.NewError(err)
	}
	to, err := sliceIndex(ectx, this, s.to, elem)
	if err != nil {
		if err != ErrSliceIndexEmpty {
			return zed.NewError(err)
		}
		n, err := elem.ContainerLength()
		if err != nil {
			panic(err)
		}
		to = int(n)
	}
	bytes := elem.Bytes
	it := bytes.Iter()
	if from < 0 {
		from = 0
	}
	for k := 0; k < to && !it.Done(); k++ {
		if k == from {
			bytes = zcode.Bytes(it)
		}
		it.Next()
	}
	return ectx.NewValue(elem.Type, bytes[:len(bytes)-len(it)])
}

package expr

import (
	"errors"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr/coerce"
	"github.com/brimdata/zed/zcode"
)

type Slice struct {
	elem  Evaluator
	from  Evaluator
	to    Evaluator
	bytes zcode.Bytes
}

func NewSlice(elem, from, to Evaluator) *Slice {
	return &Slice{
		elem: elem,
		from: from,
		to:   to,
	}
}

var ErrSliceIndex = errors.New("slice index is not a number")
var ErrSliceIndexEmpty = errors.New("slice index is empty")

func sliceIndex(slot Evaluator, elem zed.Value, rec *zed.Record) (int, error) {
	if slot == nil {
		return 0, ErrSliceIndexEmpty
	}
	zv, err := slot.Eval(rec)
	if err != nil {
		return 0, err
	}
	v, ok := coerce.ToInt(zv)
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

func (s *Slice) Eval(rec *zed.Record) (zed.Value, error) {
	elem, err := s.elem.Eval(rec)
	if err != nil {
		return elem, err
	}
	if _, ok := zed.AliasOf(elem.Type).(*zed.TypeArray); !ok {
		return zed.NewErrorf("sliced value is not an array"), nil
	}
	if elem.Bytes == nil {
		return elem, nil
	}
	from, err := sliceIndex(s.from, elem, rec)
	if err != nil && err != ErrSliceIndexEmpty {
		if err == ErrSliceIndex {
			return zed.NewError(err), nil
		}
		return zed.Value{}, err
	}
	to, err := sliceIndex(s.to, elem, rec)
	if err != nil {
		if err != ErrSliceIndexEmpty {
			if err == ErrSliceIndex {
				return zed.NewError(err), nil
			}
			return zed.Value{}, err
		}
		n, err := elem.ContainerLength()
		if err != nil {
			return zed.Value{}, err
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
		if _, _, err := it.Next(); err != nil {
			return zed.Value{}, err
		}
	}
	return zed.Value{elem.Type, bytes[:len(bytes)-len(it)]}, nil
}

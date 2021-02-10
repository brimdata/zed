package expr

import (
	"github.com/brimsec/zq/expr/coerce"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
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

func (s *Slice) Eval(rec *zng.Record) (zng.Value, error) {
	elem, err := s.elem.Eval(rec)
	if err != nil {
		return elem, err
	}
	if _, ok := zng.AliasedType(elem.Type).(*zng.TypeArray); !ok {
		return zng.NewErrorf("sliced value is not an array"), nil
	}
	if elem.Bytes == nil {
		return elem, nil
	}
	var from int
	if s.from != nil {
		zv, err := s.from.Eval(rec)
		if err != nil {
			return zv, err
		}
		v, ok := coerce.ToInt(zv)
		if !ok {
			return zng.NewErrorf("slice index is not a number"), nil
		}
		from = int(v)
	}
	var to int
	if s.to == nil {
		v, err := elem.ContainerLength()
		if err != nil {
			return zng.Value{}, err
		}
		to = int(v)
	} else {
		zv, err := s.to.Eval(rec)
		if err != nil {
			return zv, err
		}
		v, ok := coerce.ToInt(zv)
		if !ok {
			return zng.NewErrorf("slice index is not a number"), nil
		}
		to = int(v)
		if to < 0 {
			n, err := elem.ContainerLength()
			if err != nil {
				return zng.Value{}, err
			}
			to += n
		}
	}
	// XXX This could be a bit more efficient by just finding the boundary
	// in the inbound zcode.Bytes and returning the slice into that.
	// See issue #2099.
	b := s.bytes[:0]
	if b == nil {
		b = make(zcode.Bytes, 0, 100)
	}
	it := elem.Bytes.Iter()
	for k := 0; !it.Done(); k++ {
		bytes, container, err := it.Next()
		if err != nil {
			return zng.Value{}, err
		}
		if k < int(from) {
			continue
		}
		if k >= int(to) {
			break
		}
		b = zcode.AppendAs(b, container, bytes)
	}
	s.bytes = b
	return zng.Value{elem.Type, b}, nil
}

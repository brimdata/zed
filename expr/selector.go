package expr

import (
	"errors"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

type Selector struct {
	selectors []Evaluator
	cursor    []Evaluator
	iter      zcode.Iter
	iterCols  []zed.Column
	rec       *zed.Value
}

func NewSelector(selectors []Evaluator) *Selector {
	return &Selector{selectors: selectors}
}

func (s *Selector) Init(rec *zed.Value) {
	s.rec = rec
	s.cursor = s.selectors
}

func (s *Selector) Next() (zed.Value, error) {
again:
	if s.iter != nil && !s.iter.Done() {
		b, _, err := s.iter.Next()
		if err != nil {
			return zed.Value{}, err
		}
		if len(s.iterCols) == 0 {
			return zed.Value{}, errors.New("selector encountered bad record value")
		}
		typ := s.iterCols[0].Type
		s.iterCols = s.iterCols[1:]
		x := zed.Value{typ, b}
		return x, nil
	}
	if len(s.cursor) == 0 {
		return zed.Value{}, nil
	}
	zv, err := s.cursor[0].Eval(s.rec)
	if err != nil {
		return zed.Value{}, err
	}
	s.cursor = s.cursor[1:]
	if typ, ok := zv.Type.(*zed.TypeRecord); ok {
		s.iter = zv.Iter()
		s.iterCols = typ.Columns
		goto again
	}
	return zv, nil
}

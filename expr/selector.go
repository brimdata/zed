package expr

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
)

//XXX this should go away no?

type Selector struct {
	selectors []Evaluator
	cursor    []Evaluator
	iter      zcode.Iter
	iterCols  []zed.Column
	zv        *zed.Value
}

var _ Generator = (*Selector)(nil)

func NewSelector(selectors []Evaluator) *Selector {
	return &Selector{selectors: selectors}
}

func (s *Selector) Init(rec *zed.Value) {
	s.zv = rec
	s.cursor = s.selectors
}

func (s *Selector) Next(scope *Scope) *zed.Value {
again:
	if s.iter != nil && !s.iter.Done() {
		b, _, err := s.iter.Next()
		if err != nil {
			panic(err)
		}
		if len(s.iterCols) == 0 {
			panic("selector encountered more fields than present in record")
		}
		typ := s.iterCols[0].Type
		s.iterCols = s.iterCols[1:]
		return &zed.Value{typ, b}
	}
	if len(s.cursor) == 0 {
		// end of sequence (i.e., zed.Value.Type==nil)
		return &zed.Value{}
	}
	// Get the next generated value and set up the record for traversal.
	val := s.cursor[0].Eval(s.zv, scope)
	s.cursor = s.cursor[1:]
	if typ, ok := val.Type.(*zed.TypeRecord); ok {
		s.iter = val.Iter()
		s.iterCols = typ.Columns
		goto again
	}
	return val
}

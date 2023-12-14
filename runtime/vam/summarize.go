package vam

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/vector"
)

type Expr interface {
	Eval(vector.Any) vector.Any
} //XXX

type Summarize struct {
	parent Puller
	zctx   *zed.Context
	// XX Abstract this runtime into a generic table computation.
	// Then the generic interface can execute fast paths for simple scenarios.
	patterns  []AggPattern
	aggExprs  []Expr
	keyExprs  []Expr
	typeTable *zed.TypeVectorTable

	keys    []vector.Any
	vals    []vector.Any
	types   []zed.Type
	tables  map[int]aggTable
	results []aggTable
}

// XXX need keyNames, agg functions
func NewSummarize(parent Puller, patterns []AggPattern, aggNames []field.Path, aggExprs []Expr, keyNames []field.Path, keyExprs []Expr) *Summarize {
	return &Summarize{
		parent:    parent,
		patterns:  patterns,
		aggExprs:  aggExprs,
		keyExprs:  keyExprs,
		typeTable: zed.NewTypeVectorTable(),
		types:     make([]zed.Type, len(keyExprs)),
		keys:      make([]vector.Any, len(keyExprs)),
		vals:      make([]vector.Any, len(aggExprs)),
	}
}

func (s *Summarize) PullVec(done bool) (vector.Any, error) {
	if done {
		_, err := s.parent.PullVec(done)
		return nil, err
	}
	if s.results != nil {
		return s.next(), nil
	}
	for {
		//XXX check context Done
		vec, err := s.parent.PullVec(false)
		if err != nil {
			return nil, err
		}
		if vec == nil {
			for _, t := range s.tables {
				s.results = append(s.results, t)
			}
			s.tables = nil
			return s.next(), nil
		}
		// XXX the key and agg arg expressions can all be eval'd in parallel.
		types := s.types[:0]
		keys := s.keys[:0]
		for _, e := range s.keyExprs {
			v := e.Eval(vec)
			keys = append(keys, v)
			var typ zed.Type
			if v != nil {
				typ = v.Type()
			}
			types = append(types, typ)
		}
		vals := s.vals[:0]
		for _, e := range s.aggExprs {
			vals = append(vals, e.Eval(vec))
		}
		tableID := s.typeTable.Lookup(types)
		table, ok := s.tables[tableID]
		if !ok {
			table = s.newAggTable(types)
			s.tables[tableID] = table
		}
		table.update(s.keys, s.vals)
	}
}

func (s *Summarize) newAggTable(types []zed.Type) aggTable {
	return nil //XXX
}

func (s *Summarize) next() vector.Any {
	if len(s.results) == 0 {
		return nil
	}
	t := s.results[0]
	s.results = s.results[1:]
	return t.materialize()
}

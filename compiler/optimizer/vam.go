package optimizer

import (
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/pkg/field"
)

func (o *Optimizer) Vectorize(seq dag.Seq) dag.Seq {
	return walk(seq, true, func(seq dag.Seq) dag.Seq {
		if len(seq) >= 2 && isScan(seq[0]) {
			if _, ok := IsCountByString(seq[1]); ok {
				return vectorize(seq, 2)
			}
			if _, ok := IsSum(seq[1]); ok {
				return vectorize(seq, 2)
			}
		}
		return seq
	})
}

func vectorize(seq dag.Seq, n int) dag.Seq {
	return append(dag.Seq{
		&dag.Vectorize{
			Kind: "Vectorize",
			Body: seq[:n],
		},
	}, seq[n:]...)
}

func isScan(o dag.Op) bool {
	_, ok := o.(*dag.SeqScan)
	return ok
}

// IsCountByString returns whether o represents "count() by <top-level field>"
// along with the field name.
func IsCountByString(o dag.Op) (string, bool) {
	s, ok := o.(*dag.Summarize)
	if ok && len(s.Aggs) == 1 && len(s.Keys) == 1 && isCount(s.Aggs[0]) {
		return isSingleField(s.Keys[0])
	}
	return "", false
}

// IsSum return whether o represents "sum(<top-level field>)" along with the
// field name.
func IsSum(o dag.Op) (string, bool) {
	s, ok := o.(*dag.Summarize)
	if ok && len(s.Aggs) == 1 && len(s.Keys) == 0 {
		if path, ok := isSum(s.Aggs[0]); ok && len(path) == 1 {
			return path[0], true
		}
	}
	return "", false
}

func isCount(a dag.Assignment) bool {
	this, ok := a.LHS.(*dag.This)
	if !ok || len(this.Path) != 1 || this.Path[0] != "count" {
		return false
	}
	agg, ok := a.RHS.(*dag.Agg)
	return ok && agg.Name == "count" && agg.Expr == nil && agg.Where == nil
}

func isSum(a dag.Assignment) (field.Path, bool) {
	this, ok := a.LHS.(*dag.This)
	if !ok || len(this.Path) != 1 || this.Path[0] != "sum" {
		return nil, false
	}
	agg, ok := a.RHS.(*dag.Agg)
	if ok && agg.Name == "sum" && agg.Where == nil {
		return isThis(agg.Expr)
	}
	return nil, false
}

func isSingleField(a dag.Assignment) (string, bool) {
	lhs := fieldOf(a.LHS)
	rhs := fieldOf(a.RHS)
	if len(lhs) != 1 || len(rhs) != 1 || !lhs.Equal(rhs) {
		return "", false
	}
	return lhs[0], true
}

func isThis(e dag.Expr) (field.Path, bool) {
	if this, ok := e.(*dag.This); ok && len(this.Path) >= 1 {
		return this.Path, true
	}
	return nil, false
}

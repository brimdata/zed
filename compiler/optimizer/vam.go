package optimizer

import (
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/pkg/field"
)

func hackCountByString(scan *dag.SeqScan, ops []dag.Op) []dag.Op {
	if len(ops) != 2 {
		return nil
	}
	summarize, ok := ops[1].(*dag.Summarize)
	if !ok {
		return nil
	}
	if len(summarize.Aggs) != 1 {
		return nil
	}
	if ok := isCount(summarize.Aggs[0]); !ok {
		return nil
	}
	field, ok := isSingleField(summarize.Keys[0])
	if !ok {
		return nil
	}
	return []dag.Op{
		&dag.VecScan{
			Kind:  "VecScan",
			Pool:  scan.Pool,
			Paths: [][]string{{field}},
		},
		&dag.CountByStringHack{
			Kind:  "CountByStringHack",
			Field: field,
		},
		&dag.Summarize{
			Kind: "Summarize",
			Keys: []dag.Assignment{{
				Kind: "Assignment",
				LHS:  &dag.This{Kind: "This", Path: []string{field}},
				RHS:  &dag.This{Kind: "This", Path: []string{field}},
			}},
			Aggs: []dag.Assignment{{
				Kind: "Assignment",
				LHS:  &dag.This{Kind: "This", Path: []string{"count"}},
				RHS: &dag.Agg{
					Kind: "Agg",
					Name: "count",
				},
			}},
			PartialsIn: true,
		},
	}
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

func hackSum(scan *dag.SeqScan, ops []dag.Op) []dag.Op {
	if len(ops) != 3 {
		return nil
	}
	summarize, ok := ops[1].(*dag.Summarize)
	if !ok {
		return nil
	}
	if len(summarize.Aggs) != 1 {
		return nil
	}
	if len(summarize.Keys) != 0 {
		return nil
	}
	path, ok := isSum(summarize.Aggs[0])
	if !ok {
		return nil
	}
	field := path[len(path)-1] //XXX
	return []dag.Op{
		&dag.VecScan{
			Kind:  "VecScan",
			Pool:  scan.Pool,
			Paths: [][]string{path},
		},
		&dag.SumHack{
			Kind:  "SumHack",
			Field: field, //XXX
		},
		&dag.Summarize{
			Kind: "Summarize",
			Aggs: []dag.Assignment{{
				Kind: "Assignment",
				LHS:  &dag.This{Kind: "This", Path: []string{"sum"}},
				RHS: &dag.Agg{
					Kind: "Agg",
					Name: "sum",
					Expr: &dag.This{Kind: "This", Path: []string{field}},
				},
			}},
			PartialsIn: true,
		},
		ops[2],
	}
}

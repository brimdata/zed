package kernel

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/optimizer"
	"github.com/brimdata/zed/pkg/field"
	samexpr "github.com/brimdata/zed/runtime/sam/expr"
	"github.com/brimdata/zed/runtime/vam/expr"
	vamop "github.com/brimdata/zed/runtime/vam/op"
	"github.com/brimdata/zed/runtime/vam/op/summarize"
	"github.com/brimdata/zed/vector"
	"github.com/brimdata/zed/zbuf"
)

// compile compiles a DAG into a graph of runtime operators, and returns
// the leaves.
func (b *Builder) compileVam(o dag.Op, parents []vector.Puller) ([]vector.Puller, error) {
	switch o := o.(type) {
	case *dag.Fork:
		//return b.compileVecFork(o, parents)
	case *dag.Scatter:
		//return b.compileVecScatter(o, parents)
	case *dag.Scope:
		//return b.compileVecScope(o, parents)
	case *dag.Switch:
		//if o.Expr != nil {
		//	return b.compileVamExprSwitch(o, parents)
		//}
		//return b.compileVecSwitch(o, parents)
	case *dag.Join:
		// see sam version for ref
	case *dag.Merge:
		//e, err := b.compileVamExpr(o.Expr)
		//if err != nil {
		//	return nil, err
		//}
		//XXX this needs to be native
		//cmp := vamexpr.NewComparator(true, o.Order == order.Desc, e).WithMissingAsNull()
		//return []vector.Puller{vamop.NewMerge(b.rctx, parents, cmp.Compare)}, nil
	case *dag.Combine:
		//return []zbuf.Puller{combine.New(b.rctx, parents)}, nil
	default:
		var parent vector.Puller
		if len(parents) == 1 {
			parent = parents[0]
		} else if len(parents) > 1 {
			//parent = combine.New(b.rctx, parents)
			panic("TBD")
		}
		p, err := b.compileVamLeaf(o, parent)
		if err != nil {
			return nil, err
		}
		return []vector.Puller{p}, nil
	}
	return nil, fmt.Errorf("unsupported dag op in vectorize: %T", o)
}

func (b *Builder) compileVamScan(scan *dag.SeqScan, parent zbuf.Puller) (vector.Puller, error) {
	pool, err := b.lookupPool(scan.Pool)
	if err != nil {
		return nil, err
	}
	//XXX check VectorCache not nil
	return vamop.NewScanner(b.rctx, b.source.Lake().VectorCache(), parent, pool, scan.Fields, nil, nil), nil
}

func (b *Builder) compileVamLeaf(o dag.Op, parent vector.Puller) (vector.Puller, error) {
	switch o := o.(type) {
	case *dag.Cut:
		e, err := b.compileVamAssignmentsToRecordExpression(nil, o.Args)
		if err != nil {
			return nil, err
		}
		return vamop.NewYield(b.zctx(), parent, []expr.Evaluator{e}), nil
	case *dag.Filter:
		e, err := b.compileVamExpr(o.Expr)
		if err != nil {
			return nil, err
		}
		return vamop.NewFilter(b.zctx(), parent, e), nil
	case *dag.Head:
		return vamop.NewHead(parent, o.Count), nil
	case *dag.Summarize:
		// XXX This only works if the key is a string which is unknowable at
		// compile time.
		// if name, ok := optimizer.IsCountByString(o); ok {
		// 	return summarize.NewCountByString(b.zctx(), parent, name), nil
		if name, ok := optimizer.IsSum(o); ok {
			return summarize.NewSum(b.zctx(), parent, name), nil
		} else {
			return b.compileVamSummarize(o, parent)
		}
	case *dag.Pass:
		return parent, nil
	case *dag.Put:
		initial := []dag.RecordElem{
			&dag.Spread{Kind: "Spread", Expr: &dag.This{Kind: "This"}},
		}
		e, err := b.compileVamAssignmentsToRecordExpression(initial, o.Args)
		if err != nil {
			return nil, err
		}
		return vamop.NewYield(b.zctx(), parent, []expr.Evaluator{expr.NewPutter(b.zctx(), e)}), nil
	case *dag.Sort:
		b.resetResetters()
		var sortExprs []samexpr.SortEvaluator
		for _, s := range o.Args {
			k, err := b.compileExpr(s.Key)
			if err != nil {
				return nil, err
			}
			sortExprs = append(sortExprs, samexpr.NewSortEvaluator(k, s.Order))
		}
		return vamop.NewSort(b.rctx, parent, sortExprs, o.NullsFirst, o.Reverse, b.resetters), nil
	case *dag.Rename:
		srcs, dsts, err := b.compileAssignmentsToLvals(o.Args)
		if err != nil {
			return nil, err
		}
		renamer := expr.NewRenamer(b.zctx(), srcs, dsts)
		return vamop.NewYield(b.zctx(), parent, []expr.Evaluator{renamer}), nil
	case *dag.Yield:
		exprs, err := b.compileVamExprs(o.Exprs)
		if err != nil {
			return nil, err
		}
		return vamop.NewYield(b.zctx(), parent, exprs), nil
	case *dag.Output:
		// XXX Ignore Output op for vectors for now.
		return parent, nil
	default:
		return nil, fmt.Errorf("internal error: unknown dag.Op while compiling for vector runtime: %#v", o)
	}
}

func (b *Builder) compileVamAssignmentsToRecordExpression(initial []dag.RecordElem, assignments []dag.Assignment) (expr.Evaluator, error) {
	elems := initial
	for _, a := range assignments {
		lhs, ok := a.LHS.(*dag.This)
		if !ok {
			return nil, fmt.Errorf("internal error: dynamic field name not supported in vector runtime: %#v", a.LHS)
		}
		elems = append(elems, newDagRecordExprForPath(lhs.Path, a.RHS).Elems...)
	}
	return b.compileVamRecordExpr(&dag.RecordExpr{Kind: "RecordExpr", Elems: elems})
}

func newDagRecordExprForPath(path []string, expr dag.Expr) *dag.RecordExpr {
	if len(path) > 1 {
		expr = newDagRecordExprForPath(path[1:], expr)
	}
	return &dag.RecordExpr{
		Kind: "RecordExpr",
		Elems: []dag.RecordElem{
			&dag.Field{Kind: "Field", Name: path[0], Value: expr},
		},
	}
}

func (b *Builder) compileVamSeq(seq dag.Seq, parents []vector.Puller) ([]vector.Puller, error) {
	for _, o := range seq {
		var err error
		parents, err = b.compileVam(o, parents)
		if err != nil {
			return nil, err
		}
	}
	return parents, nil
}

func (b *Builder) compileVamSummarize(s *dag.Summarize, parent vector.Puller) (vector.Puller, error) {
	// compile aggs
	var aggNames []field.Path
	var aggs []*expr.Aggregator
	for _, assignment := range s.Aggs {
		aggNames = append(aggNames, assignment.LHS.(*dag.This).Path)
		agg, err := b.compileVamAgg(assignment.RHS.(*dag.Agg))
		if err != nil {
			return nil, err
		}
		aggs = append(aggs, agg)
	}
	// compile keys
	var keyNames []field.Path
	var keyExprs []expr.Evaluator
	for _, assignment := range s.Keys {
		lhs, ok := assignment.LHS.(*dag.This)
		if !ok {
			return nil, errors.New("invalid lval in groupby key")
		}
		rhs, err := b.compileVamExpr(assignment.RHS)
		if err != nil {
			return nil, err
		}
		keyNames = append(keyNames, lhs.Path)
		keyExprs = append(keyExprs, rhs)
	}
	return summarize.New(parent, b.zctx(), aggNames, aggs, keyNames, keyExprs)
}

func (b *Builder) compileVamAgg(agg *dag.Agg) (*expr.Aggregator, error) {
	name := agg.Name
	var err error
	var arg expr.Evaluator
	if agg.Expr != nil {
		arg, err = b.compileVamExpr(agg.Expr)
		if err != nil {
			return nil, err
		}
	}
	var where expr.Evaluator
	if agg.Where != nil {
		where, err = b.compileVamExpr(agg.Where)
		if err != nil {
			return nil, err
		}
	}
	return expr.NewAggregator(name, arg, where)
}

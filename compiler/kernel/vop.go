package kernel

import (
	"fmt"

	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/optimizer"
	"github.com/brimdata/zed/pkg/field"
	samexpr "github.com/brimdata/zed/runtime/sam/expr"
	vamexpr "github.com/brimdata/zed/runtime/vam/expr"
	vamop "github.com/brimdata/zed/runtime/vam/op"
	"github.com/brimdata/zed/vector"
	"github.com/brimdata/zed/zbuf"
)

// compile compiles a DAG into a graph of runtime operators, and returns
// the leaves.
func (b *Builder) compileVam(o dag.Op, parents []vector.Puller) ([]vector.Puller, error) {
	switch o := o.(type) {
	case *dag.Combine:
		//return []zbuf.Puller{combine.New(b.rctx, parents)}, nil
	case *dag.Fork:
		return b.compileVamFork(o, parents)
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
	case *dag.Scatter:
		//return b.compileVecScatter(o, parents)
	case *dag.Scope:
		//return b.compileVecScope(o, parents)
	case *dag.Switch:
		//if o.Expr != nil {
		//	return b.compileVamExprSwitch(o, parents)
		//}
		//return b.compileVecSwitch(o, parents)
	default:
		var parent vector.Puller
		if len(parents) == 1 {
			parent = parents[0]
		} else if len(parents) > 1 {
			parent = vamop.NewCombine(b.rctx, parents)
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

func (b *Builder) compileVamFork(fork *dag.Fork, parents []vector.Puller) ([]vector.Puller, error) {
	var f *vamop.Fork
	switch len(parents) {
	case 0:
		// No parents: no need for a fork since every op gets a nil parent.
	case 1:
		// Single parent: insert a fork for n-way fanout.
		f = vamop.NewFork(b.rctx, parents[0])
	default:
		// Multiple parents: insert a combine followed by a fork for n-way fanout.
		f = vamop.NewFork(b.rctx, vamop.NewCombine(b.rctx, parents))
	}
	var exits []vector.Puller
	for _, seq := range fork.Paths {
		var parent vector.Puller
		if f != nil && !isEntry(seq) {
			parent = f.AddExit()
		}
		exit, err := b.compileVamSeq(seq, []vector.Puller{parent})
		if err != nil {
			return nil, err
		}
		exits = append(exits, exit...)
	}
	return exits, nil
}

func (b *Builder) compileVamLeaf(o dag.Op, parent vector.Puller) (vector.Puller, error) {
	switch o := o.(type) {
	case *dag.Cut:
		e, err := b.compileVamAssignmentsToRecordExpression(nil, o.Args)
		if err != nil {
			return nil, err
		}
		return vamop.NewYield(b.zctx(), parent, []vamexpr.Evaluator{e}), nil
	case *dag.Drop:
		fields := make(field.List, 0, len(o.Args))
		for _, e := range o.Args {
			fields = append(fields, e.(*dag.This).Path)
		}
		dropper := vamexpr.NewDropper(b.zctx(), fields)
		return vamop.NewYield(b.zctx(), parent, []vamexpr.Evaluator{dropper}), nil
	case *dag.Filter:
		e, err := b.compileVamExpr(o.Expr)
		if err != nil {
			return nil, err
		}
		return vamop.NewFilter(b.zctx(), parent, e), nil
	case *dag.Head:
		return vamop.NewHead(parent, o.Count), nil
	case *dag.Output:
		// XXX Ignore Output op for vectors for now.
		return parent, nil
	case *dag.Over:
		return b.compileVamOver(o, parent)
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
		return vamop.NewYield(b.zctx(), parent, []vamexpr.Evaluator{vamexpr.NewPutter(b.zctx(), e)}), nil
	case *dag.Rename:
		srcs, dsts, err := b.compileAssignmentsToLvals(o.Args)
		if err != nil {
			return nil, err
		}
		renamer := vamexpr.NewRenamer(b.zctx(), srcs, dsts)
		return vamop.NewYield(b.zctx(), parent, []vamexpr.Evaluator{renamer}), nil
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
	case *dag.Summarize:
		if name, ok := optimizer.IsCountByString(o); ok {
			return vamop.NewCountByString(b.zctx(), parent, name), nil
		} else if name, ok := optimizer.IsSum(o); ok {
			return vamop.NewSum(b.zctx(), parent, name), nil
		} else {
			return nil, fmt.Errorf("internal error: unhandled dag.Summarize: %#v", o)
		}
	case *dag.Tail:
		return vamop.NewTail(parent, o.Count), nil
	case *dag.Yield:
		exprs, err := b.compileVamExprs(o.Exprs)
		if err != nil {
			return nil, err
		}
		return vamop.NewYield(b.zctx(), parent, exprs), nil
	default:
		return nil, fmt.Errorf("internal error: unknown dag.Op while compiling for vector runtime: %#v", o)
	}
}

func (b *Builder) compileVamAssignmentsToRecordExpression(initial []dag.RecordElem, assignments []dag.Assignment) (vamexpr.Evaluator, error) {
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

func (b *Builder) compileVamOver(over *dag.Over, parent vector.Puller) (vector.Puller, error) {
	// withNames, withExprs, err := b.compileDefs(over.Defs)
	// if err != nil {
	// 	return nil, err
	// }
	exprs, err := b.compileVamExprs(over.Exprs)
	if err != nil {
		return nil, err
	}
	o := vamop.NewOver(b.zctx(), parent, exprs)
	if over.Body == nil {
		return o, nil
	}
	scope := o.NewScope()
	exits, err := b.compileVamSeq(over.Body, []vector.Puller{scope})
	if err != nil {
		return nil, err
	}
	var exit vector.Puller
	if len(exits) == 1 {
		exit = exits[0]
	} else {
		// This can happen when output of over body
		// is a fork or switch.
		exit = vamop.NewCombine(b.rctx, exits)
	}
	return o.NewScopeExit(exit), nil
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

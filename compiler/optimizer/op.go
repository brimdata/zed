package optimizer

import (
	"encoding/json"

	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/order"
)

// analyzeOp returns how an input order maps to an output order based
// on the semantics of the operator.  Note that an order can go from unknown
// to known (e.g., sort) or from known to unknown (e.g., parallel paths).
// Also, when op is a Summarize operator, it's input direction (where the
// order key is presumed to be the primary group-by key) is set based
// on the layout argument.  This is clumsy and needs to change.
// See issue #2658.
func (o *Optimizer) analyzeOp(op dag.Op, layout order.Layout) (order.Layout, error) {
	// We should handle secondary keys at some point.
	// See issue #2657.
	key := layout.Primary()
	if key == nil {
		return order.Nil, nil
	}
	switch op := op.(type) {
	case *dag.Filter, *dag.Head, *dag.Pass, *dag.Uniq, *dag.Tail, *dag.Fuse, *dag.Const:
		return layout, nil
	case *dag.Cut:
		return analyzeCuts(op.Args, layout), nil
	case *dag.Pick:
		return analyzeCuts(op.Args, layout), nil
	case *dag.Drop:
		for _, f := range op.Args {
			if fieldOf(f).Equal(key) {
				return order.Nil, nil
			}
		}
		return layout, nil
	case *dag.Rename:
		for _, assignment := range op.Args {
			if fieldOf(assignment.RHS).Equal(key) {
				lhs := fieldOf(assignment.LHS)
				layout = order.NewLayout(layout.Order, field.List{lhs})
			}
		}
		return layout, nil
	case *dag.Summarize:
		return analyzeOpSummarize(op, layout), nil
	case *dag.Put:
		for _, assignment := range op.Args {
			if fieldOf(assignment.LHS).Equal(key) {
				return order.Nil, nil
			}
		}
		return layout, nil
	case *dag.Sequential:
		for _, op := range op.Ops {
			var err error
			layout, err = o.analyzeOp(op, layout)
			if err != nil {
				return order.Nil, err
			}
		}
		return layout, nil
	case *dag.Sort:
		// XXX Only single sort keys.  See issue #2657.
		if len(op.Args) != 1 {
			return order.Nil, nil
		}
		newKey := fieldOf(op.Args[0])
		if newKey == nil {
			// Not a field
			return order.Nil, nil
		}
		o := order.Asc
		if op.SortDir < 0 { // Issue #2659 change SortDir to use order.Which
			o = order.Desc
		}
		return order.NewLayout(o, field.List{key}), nil
	case *dag.From:
		var egress order.Layout
		for k := range op.Trunks {
			trunk := &op.Trunks[k]
			l, err := o.layoutOfSource(trunk.Source, layout)
			if err != nil || l.IsNil() {
				return order.Nil, err
			}
			l, err = o.analyzeOp(trunk.Seq, l)
			if err != nil {
				return order.Nil, err
			}
			if k == 0 {
				egress = l
			} else if !egress.Equal(l) {
				return order.Nil, nil
			}
		}
		return egress, nil
	default:
		return order.Nil, nil
	}
}

// summarizeOrderAndAssign determines whether its first groupby key is the
// same as the scan order or an order-preserving function thereof, and if so,
// sets ast.Summarize.InputSortDir to the propagated scan order.  It returns
// the new order (or order.Nil if unknown) that will arise after the summarize
// is applied to its input.
func analyzeOpSummarize(summarize *dag.Summarize, layout order.Layout) order.Layout {
	// Set p.InputSortDir and return true if the first grouping key
	// is inputSortField or an order-preserving function of it.
	key := layout.Keys[0]
	if len(summarize.Keys) == 0 {
		return order.Nil
	}
	groupByKey := fieldOf(summarize.Keys[0].LHS)
	if groupByKey.Equal(key) {
		rhsExpr := summarize.Keys[0].RHS
		rhs := fieldOf(rhsExpr)
		if rhs.Equal(key) || orderPreservingCall(rhsExpr, groupByKey) {
			return layout
		}
	}
	return order.Nil
}

func orderPreservingCall(e dag.Expr, key field.Path) bool {
	if call, ok := e.(*dag.Call); ok {
		switch call.Name {
		// There are probably other functions we could cover.
		// It would be good if the function declaration included
		// the info we need here.  See issue #2660.
		case "ceil", "floor", "round", "trunc":
			if len(call.Args) >= 1 && fieldOf(call.Args[0]).Equal(key) {
				return true
			}
		}
	}
	return false
}

func analyzeCuts(assignments []dag.Assignment, layout order.Layout) order.Layout {
	key := layout.Primary()
	if key == nil {
		return order.Nil
	}
	for _, a := range assignments {
		// XXX This logic (from the original parallelize code)
		// seems to have a bug.  See issue #2663.
		if !fieldOf(a.RHS).Equal(key) {
			continue
		}
		lhs := fieldOf(a.LHS)
		if lhs == nil {
			return order.Nil
		}
		if lhs.Equal(key) {
			return layout
		}
		return order.Layout{Keys: field.List{key}, Order: layout.Order}
	}
	return order.Nil
}

func fieldOf(e dag.Expr) field.Path {
	f, ok := e.(*dag.Path)
	if !ok {
		return nil
	}
	return f.Name
}

func copyOps(ops []dag.Op) []dag.Op {
	var copies []dag.Op
	for _, p := range ops {
		copies = append(copies, copyOp(p))
	}
	return copies
}

func copyOp(p dag.Op) dag.Op {
	if p == nil {
		panic("copyOp nil")
	}
	b, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	copy, err := dag.UnpackJSONAsOp(b)
	if err != nil {
		panic(err)
	}
	return copy
}

func orderSensitive(ops []dag.Op) bool {
	for _, op := range ops {
		switch op.(type) {
		case *dag.Sort, *dag.Summarize:
			return false
		case *dag.Parallel, *dag.From, *dag.Join:
			// Don't try to analyze past these operators.
			return true
		}
	}
	return true
}

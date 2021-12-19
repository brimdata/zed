package optimizer

import (
	"encoding/json"

	"github.com/brimdata/zed/compiler/ast/dag"
	astzed "github.com/brimdata/zed/compiler/ast/zed"
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
		return order.NewLayout(op.Order, field.List{key}), nil
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
	// This loop implements a very simple data flow analysis where we
	// track the known order through the scoreboard.  If on exit, there
	// is more than one field of known order, the current optimization
	// framework cannot handle this so we return unknown (order.Nil)
	// as a conservative stance to prevent any problematic optimizations.
	// If there is precisely one field of known order, then that is the
	// layout we return.  In a future version of the optimizer, we will
	// generalize this scoreboard concept across the flowgraph for a
	// comprehensive approach to dataflow analysis.  See issue #2756.
	scoreboard := make(map[string]field.Path)
	scoreboard[fieldKey(key)] = key
	for _, a := range assignments {
		lhs := fieldOf(a.LHS)
		rhs := fieldOf(a.RHS)
		if lhs == nil {
			// If we cannot statically determine the data flow,
			// we give up and return unknown.  This is overly
			// conservative in general and will miss optimization
			// opportunities, e.g., we could do dependency
			// analysis of a complex RHS expression.
			return order.Nil
		}
		lhsKey := fieldKey(lhs)
		if rhs == nil {
			// If the RHS depends on a well-defined set of fields
			// (none of which are unambiguous like this.foo[this.bar]),
			// and if all of such dependencies do not have an order
			// to preserve, then we can continue along by clearing
			// the LHS from the scoreboard knowing that is being set
			// to something that does not have a defined order.
			dependencies, ok := fieldsOf(a.RHS)
			if !ok {
				return order.Nil
			}
			for _, d := range dependencies {
				key := fieldKey(d)
				if _, ok := scoreboard[key]; ok {
					// There's a dependency on an ordered
					// field but we're not sophisticated
					// enough here to know if this preserves
					// its order...
					return order.Nil
				}
			}
			// There are no RHS dependencies on an ordered input.
			// Clobber the LHS if present from the scoreboard and continue.
			delete(scoreboard, lhsKey)
			continue
		}
		rhsKey := fieldKey(rhs)
		if _, ok := scoreboard[rhsKey]; ok {
			scoreboard[lhsKey] = lhs
			continue
		}
		if _, ok := scoreboard[lhsKey]; ok {
			// LHS is in the scoreboard and the RHS isn't, so
			// we know for sure there is no ordering guarantee
			// on the LHS field.  So clobber it and continue.
			delete(scoreboard, lhsKey)
		}
	}
	if len(scoreboard) != 1 {
		return order.Nil
	}
	for _, f := range scoreboard {
		return order.Layout{Keys: field.List{f}, Order: layout.Order}
	}
	panic("unreachable")
}

func fieldKey(f field.Path) string {
	var b []byte
	for _, s := range f {
		b = append(b, s...)
		b = append(b, 0)
	}
	return string(b)
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

func fieldsOf(e dag.Expr) (field.List, bool) {
	if e == nil {
		return nil, false
	}
	switch e := e.(type) {
	case *astzed.Primitive, *astzed.TypeValue, *dag.Search:
		return nil, true
	case *dag.Ref:
		// finish with issue #2756
		return nil, false
	case *dag.Path:
		return field.List{e.Name}, true
	case *dag.UnaryExpr:
		return fieldsOf(e.Operand)
	case *dag.BinaryExpr:
		lhs, ok := fieldsOf(e.LHS)
		if !ok {
			return nil, false
		}
		rhs, ok := fieldsOf(e.RHS)
		if !ok {
			return nil, false
		}
		return append(lhs, rhs...), true
	case *dag.Conditional:
		// finish with issue #2756
		return nil, false
	case *dag.Call:
		// finish with issue #2756
		return nil, false
	case *dag.Cast:
		return fieldsOf(e.Expr)
	case *dag.RegexpMatch:
		return fieldsOf(e.Expr)
	case *dag.RecordExpr:
		// finish with issue #2756
		return nil, false
	case *dag.ArrayExpr:
		// finish with issue #2756
		return nil, false
	case *dag.SetExpr:
		// finish with issue #2756
		return nil, false
	case *dag.MapExpr:
		// finish with issue #2756
		return nil, false
	default:
		return nil, false
	}
}

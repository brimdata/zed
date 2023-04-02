package optimizer

import (
	"encoding/json"
	"strings"

	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
)

// analyzeOp returns how an input order maps to an output order based
// on the semantics of the operator.  Note that an order can go from unknown
// to known (e.g., sort) or from known to unknown (e.g., parallel paths).
// Also, when op is a Summarize operator, it's input direction (where the
// order key is presumed to be the primary group-by key) is set based
// on the sortKey argument.  This is clumsy and needs to change.
// See issue #2658.
func (o *Optimizer) analyzeOp(op dag.Op, sortKey order.SortKey) (order.SortKey, error) {
	// We should handle secondary keys at some point.
	// See issue #2657.
	key := sortKey.Primary()
	if key == nil {
		return order.Nil, nil
	}
	switch op := op.(type) {
	case *dag.Filter, *dag.Head, *dag.Pass, *dag.Uniq, *dag.Tail, *dag.Fuse:
		return sortKey, nil
	case *dag.Cut:
		return analyzeCuts(op.Args, sortKey), nil
	case *dag.Drop:
		for _, f := range op.Args {
			if fieldOf(f).Equal(key) {
				return order.Nil, nil
			}
		}
		return sortKey, nil
	case *dag.Rename:
		for _, assignment := range op.Args {
			if fieldOf(assignment.RHS).Equal(key) {
				lhs := fieldOf(assignment.LHS)
				sortKey = order.NewSortKey(sortKey.Order, field.List{lhs})
			}
		}
		return sortKey, nil
	case *dag.Summarize:
		return analyzeOpSummarize(op, sortKey), nil
	case *dag.Put:
		for _, assignment := range op.Args {
			if fieldOf(assignment.LHS).Equal(key) {
				return order.Nil, nil
			}
		}
		return sortKey, nil
	case *dag.Sequential:
		for _, op := range op.Ops {
			var err error
			sortKey, err = o.analyzeOp(op, sortKey)
			if err != nil {
				return order.Nil, err
			}
		}
		return sortKey, nil
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
		return order.NewSortKey(op.Order, field.List{key}), nil
	case *dag.From:
		var egress order.SortKey
		for k := range op.Trunks {
			trunk := &op.Trunks[k]
			l, err := o.sortKeyOfSource(trunk.Source, sortKey)
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
func analyzeOpSummarize(summarize *dag.Summarize, sortKey order.SortKey) order.SortKey {
	// Set p.InputSortDir and return true if the first grouping key
	// is inputSortField or an order-preserving function of it.
	key := sortKey.Keys[0]
	if len(summarize.Keys) == 0 {
		return order.Nil
	}
	groupByKey := fieldOf(summarize.Keys[0].LHS)
	if groupByKey.Equal(key) {
		rhsExpr := summarize.Keys[0].RHS
		rhs := fieldOf(rhsExpr)
		if rhs.Equal(key) || orderPreservingCall(rhsExpr, groupByKey) {
			return sortKey
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
		case "bucket", "ceil", "floor", "round":
			if len(call.Args) >= 1 && fieldOf(call.Args[0]).Equal(key) {
				return true
			}
		case "every":
			return true
		}
	}
	return false
}

func analyzeCuts(assignments []dag.Assignment, sortKey order.SortKey) order.SortKey {
	key := sortKey.Primary()
	if key == nil {
		return order.Nil
	}
	// This loop implements a very simple data flow analysis where we
	// track the known order through the scoreboard.  If on exit, there
	// is more than one field of known order, the current optimization
	// framework cannot handle this so we return unknown (order.Nil)
	// as a conservative stance to prevent any problematic optimizations.
	// If there is precisely one field of known order, then that is the
	// sort key we return.  In a future version of the optimizer, we will
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
		// LHS is in the scoreboard and the RHS isn't, so
		// we know for sure there is no ordering guarantee
		// on the LHS field.  So clobber it and continue.
		delete(scoreboard, lhsKey)
	}
	if len(scoreboard) != 1 {
		return order.Nil
	}
	for _, f := range scoreboard {
		return order.SortKey{Keys: field.List{f}, Order: sortKey.Order}
	}
	panic("unreachable")
}

func fieldKey(f field.Path) string {
	return strings.Join(f, "\x00")
}

func fieldOf(e dag.Expr) field.Path {
	if this, ok := e.(*dag.This); ok {
		return this.Path
	}
	return nil
}

func copyOps(ops []dag.Op) []dag.Op {
	var copies []dag.Op
	for _, o := range ops {
		copies = append(copies, copyOp(o))
	}
	return copies
}

func copyOp(o dag.Op) dag.Op {
	if o == nil {
		panic("copyOp nil")
	}
	b, err := json.Marshal(o)
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
	case *dag.Search, *dag.Literal:
		return nil, true
	case *dag.Var:
		// finish with issue #2756
		return nil, false
	case *dag.This:
		return field.List{e.Path}, true
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

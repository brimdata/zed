package optimizer

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
)

// analyzeSortKey returns how an input order maps to an output order based
// on the semantics of the operator.  Note that an order can go from unknown
// to known (e.g., sort) or from known to unknown (e.g., conflicting parallel paths).
// Also, when op is a Summarize operator, its input direction (where the
// order key is presumed to be the primary group-by key) is set based
// on the in sort key.  This is clumsy and needs to change.
// See issue #2658.
func (o *Optimizer) analyzeSortKey(op dag.Op, in order.SortKey) (order.SortKey, error) {
	switch op := op.(type) {
	case *dag.PoolScan:
		// Ignore in and just return the sort order of the pool.
		pool, err := o.lookupPool(op.ID)
		if err != nil {
			return order.Nil, err
		}
		return pool.SortKey, nil
	case *dag.Sort:
		return sortKeyOfSort(op), nil
	}
	// We should handle secondary keys at some point.
	// See issue #2657.
	key := in.Primary()
	if key == nil {
		return order.Nil, nil
	}
	switch op := op.(type) {
	case *dag.Lister:
		// This shouldn't happen.
		return order.Nil, errors.New("internal error: dag.Lister encountered in anaylzeSortKey")
	case *dag.Filter, *dag.Head, *dag.Pass, *dag.Uniq, *dag.Tail, *dag.Fuse:
		return in, nil
	case *dag.Cut:
		return analyzeCuts(op.Args, in), nil
	case *dag.Drop:
		for _, f := range op.Args {
			if fieldOf(f).Equal(key) {
				return order.Nil, nil
			}
		}
		return in, nil
	case *dag.Rename:
		out := in
		for k := range op.Dsts {
			if fieldOf(op.Srcs[k]).Equal(key) {
				lhs := fieldOf(op.Dsts[k])
				out = order.NewSortKey(in.Order, field.List{lhs})
			}
		}
		return out, nil
	case *dag.Summarize:
		if isKeyOfSummarize(op, in) {
			return in, nil
		}
		return order.Nil, nil
	case *dag.Put:
		for _, assignment := range op.Args {
			if fieldOf(assignment.LHS).Equal(key) {
				return order.Nil, nil
			}
		}
		return in, nil
	default:
		return order.Nil, nil
	}
}

func sortKeyOfSort(op *dag.Sort) order.SortKey {
	// XXX Only single sort keys.  See issue #2657.
	if len(op.Args) != 1 {
		return order.Nil
	}
	return sortKeyOfExpr(op.Args[0], op.Order)
}

func sortKeyOfExpr(e dag.Expr, o order.Which) order.SortKey {
	key := fieldOf(e)
	if key == nil {
		return order.Nil
	}
	return order.NewSortKey(o, field.List{key})
}

// isKeyOfSummarize returns true iff its any of the groupby keys is the
// same as the given primary-key sort order or an order-preserving function
// thereof.
func isKeyOfSummarize(summarize *dag.Summarize, in order.SortKey) bool {
	if len(in.Keys) == 0 {
		return false
	}
	key := in.Keys[0]
	for _, outputKeyExpr := range summarize.Keys {
		groupByKey := fieldOf(outputKeyExpr.LHS)
		if groupByKey.Equal(key) {
			rhsExpr := outputKeyExpr.RHS
			rhs := fieldOf(rhsExpr)
			if rhs.Equal(key) || orderPreservingCall(rhsExpr, groupByKey) {
				return true
			}
		}
	}
	return false
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
	if path, ok := e.(*dag.Path); ok {
		if this := path.StaticPath(); this != nil {
			return this.Path
		}
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
	copy, err := dag.UnmarshalOp(b)
	if err != nil {
		panic(err)
	}
	return copy
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

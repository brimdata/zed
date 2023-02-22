package optimizer

import (
	"context"
	"fmt"

	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/data"
	"github.com/brimdata/zed/compiler/kernel"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
	"golang.org/x/exp/slices"
)

type Optimizer struct {
	ctx     context.Context
	entry   *dag.Sequential
	source  *data.Source
	layouts map[dag.Source]order.Layout
}

func New(ctx context.Context, entry *dag.Sequential, source *data.Source) *Optimizer {
	return &Optimizer{
		ctx:     ctx,
		entry:   entry,
		source:  source,
		layouts: make(map[dag.Source]order.Layout),
	}
}

func (o *Optimizer) Entry() *dag.Sequential {
	return o.entry
}

// MergeFilters transforms the DAG by merging adjacent filter operators so that,
// e.g., "where a | where b" becomes "where a and b".
//
// Note: MergeFilters does not descend into dag.OverExpr.Scope, so it cannot
// merge filters in "over" expressions like "sum(over a | where b | where c)".
func (o *Optimizer) MergeFilters() {
	walkOp(o.entry, func(op dag.Op) {
		if seq, ok := op.(*dag.Sequential); ok {
			// Start at the next to last element and work toward the first.
			for i := len(seq.Ops) - 2; i >= 0; i-- {
				if f1, ok := seq.Ops[i].(*dag.Filter); ok {
					if f2, ok := seq.Ops[i+1].(*dag.Filter); ok {
						// Merge the second filter into the
						// first and then delete the second.
						f1.Expr = dag.NewBinaryExpr("and", f1.Expr, f2.Expr)
						seq.Ops = slices.Delete(seq.Ops, i+1, i+2)
					}
				}
			}
		}
	})
}

func walkOp(op dag.Op, post func(dag.Op)) {
	switch op := op.(type) {
	case *dag.From:
		for _, t := range op.Trunks {
			if t.Seq != nil {
				walkOp(t.Seq, post)
			}
		}
	case *dag.Over:
		if op.Scope != nil {
			walkOp(op.Scope, post)
		}
	case *dag.Parallel:
		for _, o := range op.Ops {
			walkOp(o, post)
		}
	case *dag.Sequential:
		for _, o := range op.Ops {
			walkOp(o, post)
		}
	}
	post(op)
}

// OptimizeScan transforms the DAG by attempting to lift stateless operators
// from the downstream sequence into the trunk of each data source in the From
// operator at the entry point of the DAG.  Once these paths are lifted,
// it also attempts to move any candidate filtering operations into the
// source's pushdown predicate.  This should be called before ParallelizeScan().
// TBD: we need to do pushdown for search/cut to optimize columnar extraction.
func (o *Optimizer) OptimizeScan() error {
	if _, ok := o.entry.Ops[0].(*dag.From); !ok {
		return nil
	}
	seq := o.entry
	o.propagateScanOrder(seq, order.Nil)
	from := seq.Ops[0].(*dag.From)
	chain := seq.Ops[1:]
	layout, err := o.layoutOfFrom(from)
	if err != nil {
		return err
	}
	len, _, err := o.splittablePath(chain, layout)
	if err != nil {
		return err
	}
	if len > 0 {
		chain = chain[:len]
		for k := range from.Trunks {
			liftInto(&from.Trunks[k], copyOps(chain))
		}
		seq.Delete(1, len)
	}
	for k := range from.Trunks {
		trunk := &from.Trunks[k]
		pushDown(trunk)
		// Check to see if we can add a range pruner when the pool-key is used
		// in a normal filtering operation.
		if layout, ok := o.layouts[trunk.Source]; ok {
			if pushdown, ok := trunk.Pushdown.(*dag.Filter); ok {
				if p := newRangePruner(pushdown.Expr, layout.Primary(), layout.Order); p != nil {
					trunk.KeyPruner = p
				}
			}
		}
	}
	return nil
}

// propagateScanOrder analyzes each trunk of the From input node and
// attempts to push the scan order of the data source into the first
// downstream aggregation.  (We could continue the analysis past that
// point but don't bother yet because we do not yet support any optimization
// past the first aggregation.)  If there are multiple trunks, we only
// propagate the scan order if its the same at egress of all of the trunks.
func (o *Optimizer) propagateScanOrder(op dag.Op, parent order.Layout) (order.Layout, error) {
	switch op := op.(type) {
	case *dag.From:
		var egress order.Layout
		for k := range op.Trunks {
			trunk := &op.Trunks[k]
			l, err := o.layoutOfSource(trunk.Source, parent)
			if err != nil {
				return order.Nil, err
			}
			l, err = o.propagateScanOrder(trunk.Seq, l)
			if err != nil {
				return order.Nil, err
			}
			if k == 0 {
				egress = l
			} else if !egress.Equal(l) {
				egress = order.Nil
			}
		}
		return egress, nil
	case *dag.Summarize:
		//XXX handle only primary key for now
		key := parent.Primary()
		for _, k := range op.Keys {
			if groupByKey := fieldOf(k.LHS); groupByKey.Equal(key) {
				rhsExpr := k.RHS
				rhs := fieldOf(rhsExpr)
				if rhs.Equal(key) || orderPreservingCall(rhsExpr, groupByKey) {
					op.InputSortDir = orderAsDirection(parent.Order)
					// Currently, the groupby operator will sort its
					// output according to the primary key, but we
					// should relax this and do an analysis here as
					// to whether the sort is necessary for the
					// downstream consumer.
					return parent, nil
				}
			}
		}
		// We'll live this as unknown for now even though the groupby
		// and not try to optimize downstream of the first groupby
		// unless there is an excplicit sort encountered.
		return order.Nil, nil
	case *dag.Sequential:
		if op == nil {
			return parent, nil
		}
		for _, op := range op.Ops {
			var err error
			parent, err = o.propagateScanOrder(op, parent)
			if err != nil {
				return order.Nil, err
			}
		}
		return parent, nil
	case *dag.Parallel:
		var egress order.Layout
		for k, op := range op.Ops {
			out, err := o.propagateScanOrder(op, parent)
			if err != nil {
				return order.Nil, err
			}
			if k == 0 {
				egress = out
			} else if !egress.Equal(out) {
				egress = order.Nil
			}
		}
		return egress, nil
	case *dag.Merge:
		layout := order.NewLayout(op.Order, nil)
		if this, ok := op.Expr.(*dag.This); ok {
			layout.Keys = field.List{this.Path}
		}
		if !layout.Equal(parent) {
			layout = order.Nil
		}
		return layout, nil
	default:
		return o.analyzeOp(op, parent)
	}
}

func (o *Optimizer) layoutOfSource(s dag.Source, parent order.Layout) (order.Layout, error) {
	layout, ok := o.layouts[s]
	if !ok {
		var err error
		layout, err = o.getLayout(s, parent)
		if err != nil {
			return order.Nil, err
		}
		o.layouts[s] = layout
	}
	return layout, nil
}

func (o *Optimizer) getLayout(s dag.Source, parent order.Layout) (order.Layout, error) {
	switch s := s.(type) {
	case *dag.File:
		return s.Layout, nil
	case *dag.HTTP:
		return s.Layout, nil
	case *dag.Pool:
		return o.source.Layout(o.ctx, s), nil
	case *dag.CommitMeta:
		if s.Tap && s.Meta == "objects" {
			// For a tap into the object stream, we compile the downstream
			// DAG as if it were a normal query (so the optimizer can prune
			// objects etc.) but we execute it in the end as a meta-query.
			return o.source.Layout(o.ctx, s), nil
		}
		return order.Nil, nil
	case *dag.LakeMeta, *dag.PoolMeta:
		return order.Nil, nil
	case *dag.Pass:
		return parent, nil
	case *kernel.Reader:
		return s.Layout, nil
	default:
		return order.Nil, fmt.Errorf("unknown dag.Source type %T", s)
	}
}

// Parallelize takes a sequential operation and tries to
// parallelize it by splitting as much as possible of the sequence
// into n parallel branches. The boolean return argument indicates
// whether the flowgraph could be parallelized.
func (o *Optimizer) Parallelize(n int) error {
	replicas := n - 1
	if replicas < 1 {
		return fmt.Errorf("bad parallelization factor: %d", n)
	}
	if replicas > 50 {
		// XXX arbitrary circuit breaker
		return fmt.Errorf("parallelization factor too big: %d", n)
	}
	seq := o.entry
	from, ok := seq.Ops[0].(*dag.From)
	if !ok {
		return nil
	}
	trunks := poolTrunks(from)
	if len(trunks) == 1 {
		quietCuts(trunks[0])
		if err := o.parallelizeTrunk(seq, trunks[0], replicas); err != nil {
			return err
		}
	}
	return nil
}

func quietCuts(trunk *dag.Trunk) {
	if trunk.Seq == nil {
		return
	}
	for _, op := range trunk.Seq.Ops {
		if cut, ok := op.(*dag.Cut); ok {
			cut.Quiet = true
		}
	}
}

func poolTrunks(from *dag.From) []*dag.Trunk {
	var trunks []*dag.Trunk
	for k := range from.Trunks {
		trunk := &from.Trunks[k]
		if _, ok := trunk.Source.(*dag.Pool); ok {
			trunks = append(trunks, trunk)
		}
	}
	return trunks
}

func liftInto(trunk *dag.Trunk, branch []dag.Op) {
	if trunk.Seq == nil {
		trunk.Seq = &dag.Sequential{
			Kind: "Sequential",
		}
	}
	trunk.Seq.Ops = append(trunk.Seq.Ops, branch...)
}

func extend(trunk *dag.Trunk, op dag.Op) {
	if trunk.Seq == nil {
		trunk.Seq = &dag.Sequential{Kind: "Sequential"}
	}
	trunk.Seq.Append(op)
}

// pushDown attempts to move any filter from the front of the trunk's sequence
// into the PushDown field of the Trunk so that the runtime can push the
// filter predicate into the scanner.  This is a very simple optimization for now
// that works for only a single filter operator.  In the future, the pushown
// logic to handle arbitrary columnar operations will happen here, perhaps with
// some involvement from the DataAdaptor.
func pushDown(trunk *dag.Trunk) {
	seq := trunk.Seq
	if seq == nil || len(seq.Ops) == 0 {
		return
	}
	filter, ok := seq.Ops[0].(*dag.Filter)
	if !ok {
		return
	}
	seq.Ops = seq.Ops[1:]
	if len(seq.Ops) == 0 {
		trunk.Seq = nil
	}
	trunk.Pushdown = filter
}

// newRangePruner returns a new predicate based on the input predicate pred
// that when applied to an input value (i.e., "this") with fields from/to, returns
// true if comparisons in pred against literal values can for certain rule out
// that pred would be true for any value in the from/to range.  From/to are presumed
// to be ordered according to the order o.  This is used to prune metadata objects
// from a scan when we know the pool key range of the object could not satisfy
// the filter predicate of any of the values in the object.
func newRangePruner(pred dag.Expr, fld field.Path, o order.Which) *dag.BinaryExpr {
	lower := &dag.This{Kind: "This", Path: field.New("from")}
	upper := &dag.This{Kind: "This", Path: field.New("to")}
	if o == order.Desc {
		lower, upper = upper, lower
	}
	return buildRangePruner(pred, fld, lower, upper, o)
}

// buildRangePruner creates a DAG comparison expression that can evalaute whether
// a Zed value adhering to the from/to pattern can be excluded from a scan because
// the expression pred would evaluate to false for all values of fld in the
// from/to value range.  If a pruning decision cannot be reliably determined then
// the return value is nil.
func buildRangePruner(pred dag.Expr, fld field.Path, lower, upper *dag.This, o order.Which) *dag.BinaryExpr {
	e, ok := pred.(*dag.BinaryExpr)
	if !ok {
		// If this isn't a binary predicate composed of comparison operators, we
		// simply punt here.  This doesn't mean we can't optimize, because if the
		// unknown part (from here) appears in the context of an "and", then we
		// can still prune the known side of the "and" as implemented in the
		// logic below.
		return nil
	}
	switch e.Op {
	case "and":
		// For an "and", if we know either side is prunable, then we can prune
		// because both conditions are required.  So we "or" together the result
		// when both sub-expressions are valid.
		lhs := buildRangePruner(e.LHS, fld, lower, upper, o)
		rhs := buildRangePruner(e.RHS, fld, lower, upper, o)
		if lhs == nil {
			return rhs
		}
		if rhs == nil {
			return lhs
		}
		return dag.NewBinaryExpr("or", lhs, rhs)
	case "or":
		// For an "or", if we know both sides are prunable, then we can prune
		// because either condition is required.  So we "and" together the result
		// when both sub-expressions are valid.
		lhs := buildRangePruner(e.LHS, fld, lower, upper, o)
		rhs := buildRangePruner(e.RHS, fld, lower, upper, o)
		if lhs == nil || rhs == nil {
			return nil
		}
		return dag.NewBinaryExpr("and", lhs, rhs)
	case "==", "<", "<=", ">", ">=":
		this, literal, op := literalComparison(e)
		if this == nil || !fld.Equal(this.Path) {
			return nil
		}
		// At this point, we know we can definitely run a pruning decision based
		// on the literal value we found, the comparison op, and the lower/upper bounds.
		return rangePrunerPred(op, literal, lower, upper, o)
	default:
		return nil
	}
}

func rangePrunerPred(op string, literal *dag.Literal, lower, upper *dag.This, o order.Which) *dag.BinaryExpr {
	switch op {
	case "<":
		// key < CONST
		return compare("<=", literal, lower, o)
	case "<=":
		// key <= CONST
		return compare("<", literal, lower, o)
	case ">":
		// key > CONST
		return compare(">=", literal, upper, o)
	case ">=":
		// key >= CONST
		return compare(">", literal, upper, o)
	case "==":
		// key == CONST
		return dag.NewBinaryExpr("or",
			compare(">", lower, literal, o),
			compare("<", upper, literal, o))
	}
	panic("rangePrunerPred unknown op " + op)
}

// compare returns a DAG expression for a standard comparison operator but
// uses a call to the Zed language function "compare()" as standard comparisons
// do not handle nullsmax or cross-type comparisons (which can arise when the
// pool key value type changes).
func compare(op string, lhs, rhs dag.Expr, o order.Which) *dag.BinaryExpr {
	nullsMax := &dag.Literal{Kind: "Literal", Value: "false"}
	if o == order.Asc {
		nullsMax.Value = "true"
	}
	call := &dag.Call{Kind: "Call", Name: "compare", Args: []dag.Expr{lhs, rhs, nullsMax}}
	return dag.NewBinaryExpr(op, call, &dag.Literal{Kind: "Literal", Value: "0"})
}

func literalComparison(e *dag.BinaryExpr) (*dag.This, *dag.Literal, string) {
	switch lhs := e.LHS.(type) {
	case *dag.This:
		if rhs, ok := e.RHS.(*dag.Literal); ok {
			return lhs, rhs, e.Op
		}
	case *dag.Literal:
		if rhs, ok := e.RHS.(*dag.This); ok {
			return rhs, lhs, reverseComparator(e.Op)
		}
	}
	return nil, nil, ""
}

func reverseComparator(op string) string {
	switch op {
	case "==", "!=":
		return op
	case "<":
		return ">="
	case "<=":
		return ">"
	case ">":
		return "<="
	case ">=":
		return "<"
	}
	panic("unknown op")
}

package optimizer

import (
	"context"
	"errors"
	"fmt"

	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/data"
	"github.com/brimdata/zed/compiler/kernel"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
	"github.com/segmentio/ksuid"
	"golang.org/x/exp/slices"
)

type Optimizer struct {
	ctx     context.Context
	entry   *dag.Sequential
	sources map[*dag.Sequential]struct{}
	lake    *lake.Root
}

func New(ctx context.Context, entry *dag.Sequential, source *data.Source) *Optimizer {
	var lk *lake.Root
	if source != nil {
		lk = source.Lake()
	}
	return &Optimizer{
		ctx:     ctx,
		entry:   entry,
		sources: make(map[*dag.Sequential]struct{}),
		lake:    lk,
	}
}

func (o *Optimizer) Entry() *dag.Sequential {
	return o.entry
}

// mergeFilters transforms the DAG by merging adjacent filter operators so that,
// e.g., "where a | where b" becomes "where a and b".
//
// Note: mergeFilters does not descend into dag.OverExpr.Scope, so it cannot
// merge filters in "over" expressions like "sum(over a | where b | where c)".
func mergeFilters(op dag.Op) {
	walkOp(op, true, func(op dag.Op) {
		if seq, ok := op.(*dag.Sequential); ok {
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

func removePassOps(op dag.Op) {
	walkOp(op, true, func(op dag.Op) {
		if seq, ok := op.(*dag.Sequential); ok {
			// Start at the next to last element and work toward the first.
			for i := 0; i < len(seq.Ops); i++ {
				if _, ok := seq.Ops[i].(*dag.Pass); ok {
					seq.Ops = slices.Delete(seq.Ops, i, i+1)
					i--
					continue
				}
			}
			if len(seq.Ops) == 0 {
				seq.Ops = []dag.Op{dag.PassOp}
			}
		}
	})
}

func walkOp(op dag.Op, over bool, post func(dag.Op)) {
	switch op := op.(type) {
	case *dag.Over:
		if over && op.Scope != nil {
			walkOp(op.Scope, over, post)
		}
	case *dag.Parallel:
		for _, o := range op.Ops {
			walkOp(o, over, post)
		}
	case *dag.Sequential:
		for _, o := range op.Ops {
			walkOp(o, over, post)
		}
	}
	post(op)
}

// Optimize transforms the DAG by attempting to lift stateless operators
// from the downstream sequence into the trunk of each data source in the From
// operator at the entry point of the DAG.  Once these paths are lifted,
// it also attempts to move any candidate filtering operations into the
// source's pushdown predicate.  This should be called before ParallelizeScan().
// TBD: we need to do pushdown for search/cut to optimize columnar extraction.
func (o *Optimizer) Optimize() error {
	inlineSequentials(o.entry)
	mergeFilters(o.entry)
	o.findParPullups(o.entry)
	mergeFilters(o.entry)
	if err := o.optimizeSourcePaths(o.entry); err != nil {
		return err
	}
	removePassOps(o.entry)
	return nil
}

func (o *Optimizer) OptimizeDeleter(replicas int) error {
	inlineSequentials(o.entry)
	lister, deleter, filter := matchDeleter(o.entry.Ops)
	if lister == nil {
		return errors.New("internal error: bad deleter structure")
	}
	sortKey, err := o.sortKeyOfSource(lister)
	if err != nil {
		return err
	}
	// Check to see if we can add a range pruner when the pool key is used
	// in a normal filtering operation.
	lister.KeyPruner = maybeNewRangePruner(filter.Expr, sortKey)
	deleter.Where = filter.Expr
	chain := []dag.Op{deleter}
	par := &dag.Parallel{Kind: "Parallel", Any: true}
	for k := 0; k < replicas; k++ {
		par.Ops = append(par.Ops, &dag.Sequential{
			Kind: "Sequential",
			Ops:  copyOps(chain),
		})
	}
	var merge dag.Op
	if sortKey.IsNil() {
		merge = &dag.Combine{Kind: "Combine"}
	} else {
		// At this point, we always insert a merge as we don't know if the
		// downstream DAG requires the sort order.  A later step will look at
		// the fanin from this parallel structure and see if the merge can be
		// removed while also pushing additional ops from the output segment up into
		// the parallel branches to enhance concurrency.
		merge = &dag.Merge{
			Kind:  "Merge",
			Expr:  &dag.This{Kind: "This", Path: sortKey.Primary()},
			Order: sortKey.Order,
		}
	}
	o.entry.Ops = []dag.Op{lister, par, merge}
	return nil
}

func matchDeleter(ops []dag.Op) (*dag.Lister, *dag.Deleter, *dag.Filter) {
	if len(ops) == 3 {
		if lister, ok := ops[0].(*dag.Lister); ok {
			if deleter, ok := ops[1].(*dag.Deleter); ok {
				if filter, ok := ops[2].(*dag.Filter); ok {
					return lister, deleter, filter
				}
			}
		}
	}
	return nil, nil, nil
}

func (o *Optimizer) optimizeSourcePaths(op dag.Op) error {
	if par, ok := op.(*dag.Parallel); ok {
		for _, op := range par.Ops {
			if err := o.optimizeSourcePaths(op); err != nil {
				return err
			}
		}
		return nil
	}
	seq, ok := op.(*dag.Sequential)
	if !ok {
		return fmt.Errorf("internal error: entry point is not a source: %T", op)
	}
	if len(seq.Ops) == 0 {
		return errors.New("internal error: empty sequential operator")
	}
	if par, ok := seq.Ops[0].(*dag.Parallel); ok {
		return o.optimizeSourcePaths(par)
	}
	chain := seq.Ops[1:]
	if len(chain) == 0 {
		// Nothing to push down.
		return nil
	}
	o.propagateSortKey(seq, order.Nil)
	// See if we can lift a filtering predicate into the source op.
	// Filter might be nil in which case we just put the chain back
	// on the source op and zero out the source's filter.
	filter, chain := filterPushdown(chain)
	switch op := seq.Ops[0].(type) {
	case *dag.PoolScan:
		// Here we transform a PoolScan into a Lister followed by one or more chains
		// of slicers and sequence scanners.  We'll eventually choose other configurations
		// here based on metadata and availability of VNG.
		lister := &dag.Lister{
			Kind:   "Lister",
			Pool:   op.ID,
			Commit: op.Commit,
		}
		// Check to see if we can add a range pruner when the pool key is used
		// in a normal filtering operation.
		sortKey, err := o.sortKeyOfSource(op)
		if err != nil {
			return err
		}
		lister.KeyPruner = maybeNewRangePruner(filter, sortKey)
		seq.Ops = []dag.Op{lister}
		_, _, orderRequired, _, err := o.concurrentPath(chain, sortKey)
		if err != nil {
			return err
		}
		if orderRequired {
			seq.Ops = append(seq.Ops, &dag.Slicer{Kind: "Slicer"})
		}
		seq.Ops = append(seq.Ops, &dag.SeqScan{
			Kind:      "SeqScan",
			Pool:      op.ID,
			Filter:    filter,
			KeyPruner: lister.KeyPruner,
		})
		seq.Ops = append(seq.Ops, chain...)
		o.sources[seq] = struct{}{}
	case *dag.FileScan:
		op.Filter = filter
		seq.Ops = append([]dag.Op{op}, chain...)
	case *dag.CommitMetaScan:
		if op.Tap {
			sortKey, err := o.sortKeyOfSource(op)
			if err != nil {
				return err
			}
			// Check to see if we can add a range pruner when the pool key is used
			// in a normal filtering operation.
			op.KeyPruner = maybeNewRangePruner(filter, sortKey)
			// Delete the downstream operators when we are tapping the object list.
			seq.Ops = []dag.Op{op}
		}
	case *kernel.Reader:
		op.Filter = filter
		seq.Ops = append([]dag.Op{op}, chain...)
	case *dag.LakeMetaScan, *dag.PoolMetaScan, *dag.HTTPScan:
	default:
		return fmt.Errorf("internal error:  point to the query is not a source: %T", op)
	}
	return nil
}

// propagateSortKey analyzes each trunk of the input node
// attempts to push the scan order of the data source into the first
// downstream aggregation.  (We could continue the analysis past that
// point but don't bother yet because we do not yet support any optimization
// past the first aggregation.)  For parallel paths, we propagate
// the scan order if its the same at egress of all of the paths.
func (o *Optimizer) propagateSortKey(op dag.Op, parent order.SortKey) (order.SortKey, error) {
	switch op := op.(type) {
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
			parent, err = o.propagateSortKey(op, parent)
			if err != nil {
				return order.Nil, err
			}
		}
		return parent, nil
	case *dag.Parallel:
		var egress order.SortKey
		for k, op := range op.Ops {
			out, err := o.propagateSortKey(op, parent)
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
		sortKey := order.NewSortKey(op.Order, nil)
		if this, ok := op.Expr.(*dag.This); ok {
			sortKey.Keys = field.List{this.Path}
		}
		if !sortKey.Equal(parent) {
			sortKey = order.Nil
		}
		return sortKey, nil
	case *dag.PoolScan, *dag.Lister, *dag.SeqScan, *kernel.Reader:
		return o.sortKeyOfSource(op)
	default:
		return o.analyzeSortKey(op, parent)
	}
}

func (o *Optimizer) sortKeyOfSource(op dag.Op) (order.SortKey, error) {
	switch op := op.(type) {
	case *dag.FileScan:
		return op.SortKey, nil
	case *dag.HTTPScan:
		return op.SortKey, nil
	case *dag.PoolScan:
		return o.sortKey(op.ID)
	case *dag.Lister:
		return o.sortKey(op.Pool)
	case *dag.SeqScan:
		return o.sortKey(op.Pool)
	case *dag.CommitMetaScan:
		if op.Tap && op.Meta == "objects" {
			// For a tap into the object stream, we compile the downstream
			// DAG as if it were a normal query (so the optimizer can prune
			// objects etc.) but we execute it in the end as a meta-query.
			return o.sortKey(op.Pool)
		}
		return order.Nil, nil
	case *dag.LakeMetaScan, *dag.PoolMetaScan:
		return order.Nil, nil
	case *kernel.Reader:
		return op.SortKey, nil
	default:
		return order.Nil, fmt.Errorf("internal error: unknown source type %T", op)
	}
}

func (o *Optimizer) sortKey(id ksuid.KSUID) (order.SortKey, error) {
	pool, err := o.lookupPool(id)
	if err != nil {
		return order.Nil, err
	}
	return pool.SortKey, nil
}

func (o *Optimizer) Parallelize(n int) error {
	// Compute the number of parallel paths across all input sources to
	// achieve the desired level of concurrency.  At some point, we should
	// use a semaphore here and let each possible path use the max concurrency.
	if len(o.sources) == 0 {
		return nil
	}
	concurrency := n / len(o.sources)
	if concurrency < 2 {
		concurrency = 2
	}
	if concurrency > 50 {
		// arbitrary circuit breaker
		return fmt.Errorf("parallelization factor too big: %d", n)
	}
	for seq := range o.sources {
		lister, slicer, rest := matchSource(seq.Ops)
		// We parallelize the scanning to achieve the desired concurrency,
		// then the step below pulls downstream operators into the parallel
		// branches when possible, e.g., to parallelize aggregations etc.
		parallel, err := o.parallelizeScan(rest, concurrency)
		if err != nil {
			return err
		}
		if parallel == nil {
			// Leave the source path unmodified.
			continue
		}
		front := []dag.Op{lister}
		if slicer != nil {
			front = append(front, slicer)
		}
		// Replace the source path with the parallelized gadget.
		seq.Ops = append(front, parallel...)
	}
	o.findParPullups(o.entry)
	removePassOps(o.entry)
	return nil
}

func (o *Optimizer) lookupPool(id ksuid.KSUID) (*lake.Pool, error) {
	if o.lake == nil {
		return nil, errors.New("system error: lake operation cannot be used in non-lake context")
	}
	// This is fast because of the pool cache in the lake.
	return o.lake.OpenPool(o.ctx, id)
}

func matchSource(ops []dag.Op) (*dag.Lister, *dag.Slicer, []dag.Op) {
	lister := ops[0].(*dag.Lister)
	ops = ops[1:]
	slicer, ok := ops[0].(*dag.Slicer)
	if ok {
		ops = ops[1:]
	}
	if _, ok := ops[0].(*dag.SeqScan); !ok {
		panic("parseSource: no SeqScan")
	}
	return lister, slicer, ops
}

// filterPushdown attempts to move any filter from the front of op chain
// and returns the filter's expression (and the modified chain) so that
// the runtime can push the filter predicate into the scanner.
func filterPushdown(in []dag.Op) (dag.Expr, []dag.Op) {
	if len(in) == 0 {
		return nil, in
	}
	filter, ok := in[0].(*dag.Filter)
	if !ok {
		return nil, in
	}
	return filter.Expr, in[1:]
}

// newRangePruner returns a new predicate based on the input predicate pred
// that when applied to an input value (i.e., "this") with fields from/to, returns
// true if comparisons in pred against literal values can for certain rule out
// that pred would be true for any value in the from/to range.  From/to are presumed
// to be ordered according to the order o.  This is used to prune metadata objects
// from a scan when we know the pool key range of the object could not satisfy
// the filter predicate of any of the values in the object.
func newRangePruner(pred dag.Expr, fld field.Path, o order.Which) dag.Expr {
	min := &dag.This{Kind: "This", Path: field.New("min")}
	max := &dag.This{Kind: "This", Path: field.New("max")}
	if e := buildRangePruner(pred, fld, min, max); e != nil {
		return e
	}
	return nil
}

func maybeNewRangePruner(pred dag.Expr, sortKey order.SortKey) dag.Expr {
	if !sortKey.IsNil() && pred != nil {
		return newRangePruner(pred, sortKey.Primary(), sortKey.Order)
	}
	return nil
}

// buildRangePruner creates a DAG comparison expression that can evalaute whether
// a Zed value adhering to the from/to pattern can be excluded from a scan because
// the expression pred would evaluate to false for all values of fld in the
// from/to value range.  If a pruning decision cannot be reliably determined then
// the return value is nil.
func buildRangePruner(pred dag.Expr, fld field.Path, min, max *dag.This) *dag.BinaryExpr {
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
		lhs := buildRangePruner(e.LHS, fld, min, max)
		rhs := buildRangePruner(e.RHS, fld, min, max)
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
		lhs := buildRangePruner(e.LHS, fld, min, max)
		rhs := buildRangePruner(e.RHS, fld, min, max)
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
		return rangePrunerPred(op, literal, min, max)
	default:
		return nil
	}
}

func rangePrunerPred(op string, literal *dag.Literal, min, max *dag.This) *dag.BinaryExpr {
	switch op {
	case "<":
		// key < CONST
		return compare("<=", literal, min)
	case "<=":
		// key <= CONST
		return compare("<", literal, min)
	case ">":
		// key > CONST
		return compare(">=", literal, max)
	case ">=":
		// key >= CONST
		return compare(">", literal, max)
	case "==":
		// key == CONST
		return dag.NewBinaryExpr("or",
			compare(">", min, literal),
			compare("<", max, literal))
	}
	panic("rangePrunerPred unknown op " + op)
}

// compare returns a DAG expression for a standard comparison operator but
// uses a call to the Zed language function "compare()" as standard comparisons
// do not handle nullsmax or cross-type comparisons (which can arise when the
// pool key value type changes).
func compare(op string, lhs, rhs dag.Expr) *dag.BinaryExpr {
	nullsMax := &dag.Literal{Kind: "Literal", Value: "true"}
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

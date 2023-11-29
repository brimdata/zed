package optimizer

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/order"
)

// XXX Remove this and use native order.Direction in group-by.  See Issue #4505.
func orderAsDirection(which order.Which) int {
	if which == order.Asc {
		return 1
	}
	return -1
}

func (o *Optimizer) parallelizeScan(ops []dag.Op, replicas int) ([]dag.Op, error) {
	// For now we parallelize only pool scans and no metadata scans.
	// We can do the latter when we want to scale the performance of metadata.
	if replicas < 2 {
		return nil, fmt.Errorf("internal error: parallelizeScan: bad replicas: %d", replicas)
	}
	if scan, ok := ops[0].(*dag.SeqScan); ok {
		return o.parallelizeSeqScan(scan, ops, replicas)
	}
	return nil, errors.New("parallelization of non-pool queries is not yet supported")
}

func (o *Optimizer) parallelizeSeqScan(scan *dag.SeqScan, ops []dag.Op, replicas int) ([]dag.Op, error) {
	if len(ops) == 1 && scan.Filter == nil {
		// We don't try to parallelize the path if it's simply scanning and does no
		// other work.  We might want to revisit this down the road if
		// the system would benefit for parallel reading and merging.
		return nil, nil
	}
	srcSortKey, err := o.sortKeyOfSource(scan)
	if err != nil {
		return nil, err
	}
	if len(srcSortKey.Keys) > 1 {
		// XXX Don't yet support multi-key ordering.  See Issue #2657.
		return nil, nil
	}
	// concurrentPath will check that the path consisting of the original source
	// sequence and any lifted sequence is still parallelizable.
	n, outputKey, _, needMerge, err := o.concurrentPath(ops[1:], srcSortKey)
	if err != nil {
		return nil, err
	}
	// XXX Fix this to handle multi-key merge. See Issue #2657.
	if len(outputKey.Keys) > 1 {
		return nil, nil
	}
	head := ops[:n+1]
	tail := ops[n+1:]
	scatter := &dag.Scatter{
		Kind:  "Scatter",
		Paths: make([]dag.Seq, replicas),
	}
	for k := 0; k < replicas; k++ {
		scatter.Paths[k] = copyOps(head)
	}
	var merge dag.Op
	if needMerge {
		// At this point, we always insert a merge as we don't know if the
		// downstream DAG requires the sort order.  A later step will look at
		// the fanin from this parallel structure and see if the merge can be
		// removed while also pushing additional ops from the output segment up into
		// the parallel branches to enhance concurrency.
		merge = &dag.Merge{
			Kind:  "Merge",
			Expr:  &dag.This{Kind: "This", Path: outputKey.Primary()},
			Order: outputKey.Order,
		}
	} else {
		merge = &dag.Combine{Kind: "Combine"}
	}
	return append([]dag.Op{scatter, merge}, tail...), nil
}

func (o *Optimizer) optimizeParallels(seq dag.Seq) {
	walk(seq, false, func(seq dag.Seq) dag.Seq {
		for ops := seq; len(ops) >= 3; ops = ops[1:] {
			o.liftIntoParPaths(ops)
		}
		return seq
	})
}

// liftIntoParPaths examines a sequence of Ops to see if there's an opportunity to
// lift operations downstream from a parallel Op into its parallel paths to
// enhance concurrency.  If so, we modify the sequential ops in place.
func (o *Optimizer) liftIntoParPaths(ops []dag.Op) {
	if len(ops) < 2 {
		// Need a parallel, an optional merge/combine, and something downstream.
		return
	}
	// We should optimize dag.Fork in the same way.  See issue #4559.
	fork, ok := ops[0].(*dag.Scatter)
	if !ok {
		return
	}
	egress := 1
	var merge *dag.Merge
	switch op := ops[1].(type) {
	case *dag.Merge:
		merge = op
		egress = 2
	case *dag.Combine:
		egress = 2
	}
	if egress >= len(ops) {
		return
	}
	switch op := ops[egress].(type) {
	case *dag.Summarize:
		// To decompose the groupby, we split the flowgraph into branches that run up to and including a groupby,
		// followed by a post-merge groupby that composes the results.
		// Copy the aggregator into the tail of the trunk and arrange
		// for partials to flow between them.
		if op.PartialsIn || op.PartialsOut {
			// Need an unmodified summarize to split into its parials pieces.
			return
		}
		for k := range fork.Paths {
			partial := copyOp(op).(*dag.Summarize)
			partial.PartialsOut = true
			fork.Paths[k].Append(partial)
		}
		op.PartialsIn = true
		// The upstream aggregators will compute any key expressions
		// so the ingress aggregator should simply reference the key
		// by its name.  This loop updates the ingress to do so.
		for k := range op.Keys {
			op.Keys[k].RHS = op.Keys[k].LHS
		}
	case *dag.Sort:
		if len(op.Args) != 1 {
			return
		}
		if merge != nil {
			mergeKey := sortKeyOfExpr(merge.Expr, merge.Order)
			if mergeKey.IsNil() {
				// If the merge expression isn't a field, don't try to pull it up.
				// XXX We could generalize this to test for equal expressions by
				// doing an expression comparison. See issue #4524.
				return
			}
			sortKey := sortKeyOfSort(op)
			if !sortKey.Equal(mergeKey) {
				return
			}
		}
		for k := range fork.Paths {
			fork.Paths[k].Append(copyOp(op))
		}
		if merge == nil {
			merge = &dag.Merge{
				Kind:  "Merge",
				Expr:  op.Args[0],
				Order: op.Order,
			}
			if egress == 2 {
				ops[1] = merge
				ops[2] = dag.PassOp
			} else {
				ops[egress] = merge
			}
		} else {
			// There already was an appropriate merge.
			// Smash the sort into a nop.
			ops[egress] = dag.PassOp
		}
	case *dag.Head, *dag.Tail:
		// Copy the head or tail into the parallel path and leave the original in
		// place which will apply another head or tail after the merge.
		for k := range fork.Paths {
			fork.Paths[k].Append(copyOp(op))
		}
	case *dag.Cut, *dag.Drop, *dag.Put, *dag.Rename, *dag.Filter:
		if merge != nil {
			// See if this op would disrupt the merge-on key
			mergeKey, err := o.propagateSortKeyOp(merge, []order.SortKey{order.Nil})
			if err != nil || mergeKey[0].IsNil() {
				// Bail if there's a merge with a non-key expression.
				return
			}
			key, err := o.propagateSortKeyOp(op, mergeKey)
			if err != nil || !key[0].Equal(mergeKey[0]) {
				// This operator destroys the merge order so we cannot
				// lift it up into the parallel legs in front of the merge.
				return
			}
		}
		for k := range fork.Paths {
			fork.Paths[k].Append(copyOp(op))
		}
		// this will get removed later
		ops[egress] = dag.PassOp
	}
}

// concurrentPath returns the largest path within ops from front to end that can
// be parallelized and run concurrently while preserving its semantics where
// the input to ops is known to have an order defined by sortKey (or order.Nil
// if unknown).
// The length of the concurrent path is returned and the sort order at
// exit from that path is returned.  If sortKey is zero, then the
// concurrent path is allowed to include operators that do not guarantee
// an output order.
func (o *Optimizer) concurrentPath(ops []dag.Op, sortKey order.SortKey) (int, order.SortKey, bool, bool, error) {
	for k := range ops {
		switch op := ops[k].(type) {
		// This should be a boolean in op.go that defines whether
		// function can be parallelized... need to think through
		// what the meaning is here exactly.  This is all still a bit
		// of a heuristic.  See #2660 and #2661.
		case *dag.Summarize:
			// We want input sorted when we are preserving order into
			// group-by so we can release values incrementally which is really
			// important when doing a head on the group-by results
			if isKeyOfSummarize(op, sortKey) {
				// Keep the input ordered so we can incrementally release
				// results from the groupby as a streaming operation.
				return k, sortKey, true, true, nil
			}
			return k, order.Nil, false, false, nil
		case *dag.Sort:
			newKey := sortKeyOfSort(op)
			if newKey.IsNil() {
				// No analysis for sort without expression since we can't
				// parallelize the heuristic.  We should revisit these semantics
				// and define a global order across Zed type.
				return 0, order.Nil, false, false, nil
			}
			return k, newKey, false, true, nil
		case *dag.Load:
			return k, order.Nil, true, true, nil
		case *dag.Fork, *dag.Scatter, *dag.Head, *dag.Tail, *dag.Uniq, *dag.Fuse, *dag.Join:
			return k, sortKey, true, true, nil
		default:
			next, err := o.analyzeSortKey(op, sortKey)
			if err != nil {
				return 0, order.Nil, false, false, err
			}
			if !sortKey.IsNil() && next.IsNil() {
				return k, sortKey, true, true, nil
			}
			sortKey = next
		}
	}
	return len(ops), sortKey, true, true, nil
}

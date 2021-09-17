package optimizer

import (
	"errors"

	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/order"
)

func orderAsDirection(which order.Which) int {
	if which == order.Asc {
		return 1
	}
	return -1
}

//XXX assume the trunk is from a from op at seq.Ops[0] and we will
// possible insert an operator at seq.Op[1]
func (o *Optimizer) parallelizeTrunk(seq *dag.Sequential, trunk *dag.Trunk, replicas int) error {
	from, ok := seq.Ops[0].(*dag.From)
	if !ok {
		return errors.New("internal error: parallelizeTrunk: entry is not a From")
	}
	if len(from.Trunks) > 1 {
		// No support for multi-trunk from's yet, which only arise for
		// joins and other peculaliar mixins of different sources.
		// We need to handle join parallelization differently, though
		// the logic is not very far from what's here.
		return nil
	}
	if len(from.Trunks) == 0 {
		return errors.New("internal error: no trunks in dag.From")
	}
	egressLayout, err := o.layoutOfTrunk(trunk)
	if err != nil {
		return err
	}
	if len(egressLayout.Keys) > 1 {
		// XXX don't yet support multi-key ordering
		return nil
	}
	// This logic requires that there is only one trunk in the From,
	// as checked above.
	layout, err := o.layoutOfSource(trunk.Source, order.Nil)
	if err != nil {
		return err
	}
	// Check that the path consisting of the original from
	// sequence and any lifted sequence is still parallelizable.
	if trunk.Seq != nil && len(trunk.Seq.Ops) > 0 {
		n, newLayout, err := o.splittablePath(trunk.Seq.Ops, layout)
		if err != nil {
			return err
		}
		if n != len(trunk.Seq.Ops) {
			return nil
		}
		// If the trunk operators affect the scan layout, then update
		// it here so the merge will properly happen below...
		layout = newLayout
	}
	if len(seq.Ops) < 2 {
		// There are no operators past the trunk.  Just parallelize
		// and merge the trunk here.
		if err := insertMerge(seq, layout); err != nil {
			return err
		}
		replicateTrunk(from, trunk, replicas)
		return nil
	}
	switch ingress := seq.Ops[1].(type) {
	case *dag.Join, *dag.Parallel, *dag.Sequential:
		return nil
	case *dag.Summarize:
		// To decompose the groupby, we split the flowgraph into branches that run up to and including a groupby,
		// followed by a post-merge groupby that composes the results.
		// Copy the aggregator into the tail of the trunk and arrange
		// for partials to flow between them.
		egress := copyOp(ingress).(*dag.Summarize)
		egress.PartialsOut = true
		ingress.PartialsIn = true
		// The upstream aggregators will compute any key expressions
		// so the ingress aggregator should simply reference the key
		// by its name.  This loop updates the ingress to do so.
		keys := ingress.Keys
		for k := range keys {
			keys[k].RHS = keys[k].LHS
		}
		extend(trunk, egress)
		seq.Ops[1] = ingress
		//
		// Add a merge-by if this a streaming every aggregator.
		//
		if ingress.Duration != nil {
			// We insert a mergeby ts in front of the partialsIn aggregator.
			// If the inbound layout doesn't match up here then the
			// every operator won't work right so we flag
			// a compilation error..., e.g., it's like saying
			//   * | sort x | every 1h count() by _path
			// Actually, this could work it just can't stream the
			// every's as they finish.  We should work out this logic.
			// Otherwise, the runtime builder will insert a simple combiner.
			// Note: combiner has a nice flow-control feature in that it
			// allows a fast upstream send to complete without HOL blocking
			// while the slow guys continue on.  See Issue #2662.
			if !layout.Primary().Equal(field.New("ts")) {
				return errors.New("aggregation requiring 'every' semantics requires input sorted by 'ts'")
			}
			if err := insertMerge(seq, layout); err != nil {
				return err
			}
		}
		replicateTrunk(from, trunk, replicas)
		return nil
	case *dag.Sort:
		if len(ingress.Args) > 1 {
			// Unknown or multiple sort fields: we sort after
			// the merge, which can be unordered.
			replicateTrunk(from, trunk, replicas)
			return nil
		}
		// Single sort field: we can sort in each parallel branch,
		// and then do an ordered merge.
		var mergeKey field.Path
		if len(ingress.Args) > 0 {
			mergeKey = fieldOf(ingress.Args[0])
			if mergeKey == nil {
				// Sort key is an expression instead of a
				// field.  Don't try to sort.
				// XXX This would be parallelizable if we
				// did on merge on the same expression.
				return nil
			}
		}
		if mergeKey == nil {
			// No sort key.  We can parallelize the trunk
			// but we don't lift the heuristic sort into the
			// trunk since we don't know what to merge on since
			// merge does not have the sort key heuristic.
			// Also, we don't pass a merge order here since the
			// ingress sort doesn't care.
			return replicateAndMerge(seq, order.Nil, from, trunk, replicas)
		}
		// Lift sort into trunk for replication, delete the sort from
		// the main sequence, then add back a merge to effect a merge sort.
		extend(trunk, ingress)
		seq.Delete(1, 1)
		layout := order.NewLayout(ingress.Order, field.List{mergeKey})
		return replicateAndMerge(seq, layout, from, trunk, replicas)
	case *dag.Head, *dag.Tail:
		if layout.IsNil() {
			// Unknown order: we can't parallelize because we can't maintain this unknown order at the merge point.
			return nil
		}
		// Copy a head/tail into the trunk and leave the original in
		// place which will apply another head/tail after the merge.
		egress := copyOp(ingress)
		extend(trunk, egress)
		return replicateAndMerge(seq, layout, from, trunk, replicas)
	case *dag.Cut, *dag.Pick, *dag.Drop, *dag.Put, *dag.Rename:
		//XXX shouldn't these check for mergeKey = nil?
		return replicateAndMerge(seq, layout, from, trunk, replicas)
	default:
		// If we're here, we reached the end of the flowgraph without
		// coming across a merge-forcing proc. If inputs are sorted,
		// we can parallelize the entire chain and do an ordered
		// merge. Otherwise, no parallelization.
		if layout.IsNil() {
			// Unknown order: we can't parallelize because
			// we can't maintain this unknown order at the merge point,
			// but we shouldn't care if the client doesn't care
			// so this goes back to whether the scan order was
			// specified in the source. See Issue #2661.
			return nil
		}
		return replicateAndMerge(seq, layout, from, trunk, replicas)
	}
}

func replicateAndMerge(seq *dag.Sequential, layout order.Layout, from *dag.From, trunk *dag.Trunk, replicas int) error {
	if err := insertMerge(seq, layout); err != nil {
		return err
	}
	replicateTrunk(from, trunk, replicas)
	return nil
}

func insertMerge(seq *dag.Sequential, layout order.Layout) error {
	// layout represents the order we need to preserve at exit from the
	// parallel paths within the trunk.  It is nil if the order implied
	// by the DAG is unknown, meaning we do not need to preserve it and
	// thus do not need to insert a merge operator (resulting in the runtime
	// building a more efficient combine operator).
	if layout.IsNil() {
		return nil
	}
	// XXX Fix this to handle multi-key merge. See Issue #2657.
	head := []dag.Op{seq.Ops[0], &dag.Merge{
		Kind:  "Merge",
		Key:   layout.Primary(),
		Order: layout.Order,
	}}
	seq.Ops = append(head, seq.Ops[1:]...)
	return nil
}

func replicateTrunk(from *dag.From, trunk *dag.Trunk, replicas int) {
	// We use the same source pointer across the replicas.  This is very
	// important as the runtime uses pointer equivalence here to determine
	// that multiple trunks are sharing the same scan so that scan concurrency
	// will be correctly realized.
	src := trunk.Source
	seq := trunk.Seq
	if seq != nil && len(seq.Ops) == 0 {
		seq = nil
	}
	for k := 0; k < replicas; k++ {
		var newSeq *dag.Sequential
		if seq != nil {
			newSeq = &dag.Sequential{
				Kind: "Sequential",
				Ops:  copyOps(seq.Ops),
			}
		}
		replica := dag.Trunk{
			Kind:   "Trunk",
			Source: src,
			Seq:    newSeq,
		}
		from.Trunks = append(from.Trunks, replica)
	}
}

func (o *Optimizer) layoutOfFrom(from *dag.From) (order.Layout, error) {
	layout, err := o.layoutOfTrunk(&from.Trunks[0])
	if err != nil {
		return order.Nil, err
	}
	for k := range from.Trunks[1:] {
		next, err := o.layoutOfTrunk(&from.Trunks[k])
		if err != nil || !next.Equal(layout) {
			return order.Nil, err
		}
	}
	return layout, nil
}

func (o *Optimizer) layoutOfTrunk(trunk *dag.Trunk) (order.Layout, error) {
	layout, err := o.layoutOfSource(trunk.Source, order.Nil)
	if err != nil {
		return order.Nil, err
	}
	if trunk.Seq == nil {
		return layout, nil
	}
	if trunk.Pushdown != nil {
		layout, err = o.analyzeOp(trunk.Pushdown, layout)
		if err != nil {
			return order.Nil, err
		}
	}
	return o.analyzeOp(trunk.Seq, layout)
}

// splittablePath returns the largest path within ops from front to end that is splittable.
// The length of the splittablePath path is returned and the stream order at
// exit from that path is returned.  If layout is zero, then the
// splittable path is allowed to include operators that do not guarantee
// a stream order.  The property of the returned path is that it may be
// executed in parallel with some way to merge of the the parallel results.
// The layout parameter defines the input layout.
func (o *Optimizer) splittablePath(ops []dag.Op, layout order.Layout) (int, order.Layout, error) {
	requireOrder := !layout.IsNil()
	if requireOrder {
		// If the input stream is ordered, then we will preserve order,
		// but only if we have to because there are nor order-destructive ops.
		// Note we could be smarter here by taking into account what is
		// in the upstream trunk, but for now, we keep it simple.  Later
		// we will back propagate the output order desired (e.g., by noting
		// when the user requests a sort at the output stage) and use
		// this information to inform the scan decision (i.e., doing a more
		// efficient unordered scan when the scan order isn't necessary).
		// See issue #2661.
		requireOrder = orderSensitive(ops)
	}
	for k := range ops {
		switch op := ops[k].(type) {
		// This should be a boolean in op.go that defines whether
		// function can be parallelized... need to think through
		// what the meaning is here exactly.  This is all still a bit
		// of a heuristic.  See #2660 and #2661.
		case *dag.Summarize, *dag.Sort, *dag.Parallel, *dag.Head, *dag.Tail, *dag.Uniq, *dag.Fuse, *dag.Sequential, *dag.Join:
			return k, layout, nil
		default:
			next, err := o.analyzeOp(op, layout)
			if err != nil {
				return 0, order.Nil, err
			}
			if requireOrder && next.IsNil() {
				return k, layout, nil
			}
			layout = next
		}
	}
	return len(ops), layout, nil
}

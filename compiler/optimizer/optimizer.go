package optimizer

import (
	"context"
	"errors"
	"fmt"

	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/ast/zed"
	"github.com/brimdata/zed/compiler/kernel"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/proc"
)

type Optimizer struct {
	ctx     context.Context
	entry   *dag.Sequential
	adaptor proc.DataAdaptor
	layouts map[dag.Source]order.Layout
}

func New(ctx context.Context, entry *dag.Sequential, adaptor proc.DataAdaptor) (*Optimizer, error) {
	if _, ok := entry.Ops[0].(*dag.From); !ok {
		return nil, errors.New("DAG entry point is not a 'from' operator")
	}
	return &Optimizer{
		ctx:     ctx,
		entry:   entry,
		adaptor: adaptor,
		layouts: make(map[dag.Source]order.Layout),
	}, nil
}

func (o *Optimizer) Entry() *dag.Sequential {
	return o.entry
}

// OptimizeScan transforms the DAG by attempting to lift stateless operators
// from the downstream sequence into the trunk of each data source in the From
// operator at the entry point of the DAG.  Once these paths are lifted,
// it also attempts to move any candidate filtering operations into the
// source's pushdown predicate.  This should be called before ParallelizeScan().
// TBD: we need to do pushdown for search/cut to optimize columnar extraction.
func (o *Optimizer) OptimizeScan() error {
	seq := o.entry
	o.propagateScanOrder(seq, order.Nil)
	from := seq.Ops[0].(*dag.From)
	chain := seq.Ops[1:]
	layout, err := o.layoutOfFrom(from)
	if err != nil {
		return err
	}
	len, layout, err := o.splittablePath(chain, layout)
	if err != nil {
		return err
	}
	if len > 0 {
		chain = chain[:len]
		for k := range from.Trunks {
			trunk := &from.Trunks[k]
			liftInto(trunk, copyOps(chain))
			pushDown(trunk)
		}
		seq.Delete(1, len)
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
		if len(op.Keys) > 0 {
			groupByKey := fieldOf(op.Keys[0].LHS)
			if groupByKey.Equal(key) {
				rhsExpr := op.Keys[0].RHS
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
		layout := order.NewLayout(op.Order, field.List{op.Key})
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
	if pool, ok := s.(*dag.Pool); ok {
		scanOrder, _ := order.ParseDirection(pool.ScanOrder)
		// If the requested scan order is the same order as the pool,
		// then we can use it.  Otherwise, the scan is going against
		// the grain and we don't yet have the logic to reverse the
		// scan of each object, though this should be relatively
		// easy to add.  See issue #2665.
		if scanOrder != order.Unknown && !scanOrder.HasOrder(layout.Order) {
			layout = order.Nil
		}
	}
	return layout, nil
}

func (o *Optimizer) getLayout(s dag.Source, parent order.Layout) (order.Layout, error) {
	switch s := s.(type) {
	case *dag.File:
		return s.Layout, nil
	case *dag.HTTP:
		return s.Layout, nil
	case *dag.Pool, *dag.LakeMeta, *dag.PoolMeta, *dag.CommitMeta:
		return o.adaptor.Layout(o.ctx, s), nil
	case *dag.Pass:
		return parent, nil
	case *kernel.Reader:
		return s.Layout, nil
	default:
		return order.Nil, fmt.Errorf("unknown dag.Source type %T", s)
	}
}

// Parallelize takes a sequential proc AST and tries to
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
	from := seq.Ops[0].(*dag.From)
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
	trunk.Pushdown.Scan = filter
	if e := indexFilterExpr(filter.Expr); e != nil {
		trunk.Pushdown.Index = &dag.Filter{
			Kind: "Filter",
			Expr: indexFilterExpr(filter.Expr),
		}
	}
}

// indexFilterExpr returns a watered down version of the Scan Filter expression
// that can be digested by index.  All parts of the expressions tree are removed
// that are not:
// - An '=', '>', '>=', '<', '<=', 'and' or 'or' BinaryExpr
// - Leaf BinaryExprs with the LHS of *dag.Path and RHS of *zed.Primitive
func indexFilterExpr(node dag.Expr) dag.Expr {
	e, ok := node.(*dag.BinaryExpr)
	if !ok {
		return nil
	}
	switch e.Op {
	case "and", "or":
		lhs, rhs := indexFilterExpr(e.LHS), indexFilterExpr(e.RHS)
		if lhs == nil {
			return rhs
		} else if rhs == nil {
			return lhs
		}
		return &dag.BinaryExpr{
			Kind: "BinaryExpr",
			Op:   e.Op,
			LHS:  lhs,
			RHS:  rhs,
		}
	case "=", ">", ">=", "<", "<=":
		_, rok := e.RHS.(*zed.Primitive)
		_, lok := e.LHS.(*dag.Path)
		if lok && rok {
			return &dag.BinaryExpr{
				Kind: "BinaryExpr",
				Op:   e.Op,
				LHS:  e.LHS,
				RHS:  e.RHS,
			}
		}
	}
	return nil
}

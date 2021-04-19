package optimizer

import (
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/field"
)

type Optimizer struct {
	unopt   dag.Op
	entry   dag.Op
	filter  dag.Expr
	sortKey field.Static
	sortRev bool
}

func New(op dag.Op) *Optimizer {
	return &Optimizer{
		unopt: op,
		entry: op,
	}
}

func (o *Optimizer) Entry() dag.Op {
	return o.entry
}

func (o *Optimizer) Filter() dag.Expr {
	return o.filter
}

func (o *Optimizer) Optimize() error {
	// Currently, we only lift the filter into the buffer scanner
	// and push it out to distributed workers.  This could also be
	// used by ZST to read only the columns but we need a bit more
	// data structure here.  Also, this is where we would push

	// analytics and search|cut into the column reader.
	o.filter, o.entry = liftFilter(o.entry)
	return nil
}

func (o *Optimizer) SetInputOrder(sortKey field.Static, sortRev bool) {
	o.sortKey = sortKey
	o.sortRev = sortRev
	SetGroupByProcInputSortDir(o.entry, sortKey, zbufDirInt(sortRev))
}

// IsParallelizable reports whether Parallelize can parallelize p when called
// with the same arguments.
func (o *Optimizer) IsParallelizable() bool {
	_, ok := parallelize(copyOp(o.unopt), 0, o.sortKey, o.sortRev)
	return ok
}

// Parallelize takes a sequential proc AST and tries to
// parallelize it by splitting as much as possible of the sequence
// into n parallel branches. The boolean return argument indicates
// whether the flowgraph could be parallelized.
func (o *Optimizer) Parallelize(n int) bool {
	op, ok := parallelize(copyOp(o.unopt), n, o.sortKey, o.sortRev)
	if ok {
		o.entry = op
	}
	return ok
}

package kernel

import (
	"context"
	"errors"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/proc/combine"
	"github.com/brimdata/zed/proc/explode"
	"github.com/brimdata/zed/proc/exprswitch"
	"github.com/brimdata/zed/proc/from"
	"github.com/brimdata/zed/proc/fuse"
	"github.com/brimdata/zed/proc/head"
	"github.com/brimdata/zed/proc/join"
	"github.com/brimdata/zed/proc/merge"
	"github.com/brimdata/zed/proc/pass"
	"github.com/brimdata/zed/proc/put"
	"github.com/brimdata/zed/proc/shape"
	"github.com/brimdata/zed/proc/sort"
	"github.com/brimdata/zed/proc/split"
	"github.com/brimdata/zed/proc/switcher"
	"github.com/brimdata/zed/proc/tail"
	"github.com/brimdata/zed/proc/top"
	"github.com/brimdata/zed/proc/traverse"
	"github.com/brimdata/zed/proc/uniq"
	"github.com/brimdata/zed/proc/yield"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
)

var ErrJoinParents = errors.New("join requires two upstream parallel query paths")

type Builder struct {
	pctx       *proc.Context
	adaptor    proc.DataAdaptor
	schedulers map[dag.Source]proc.Scheduler
}

func NewBuilder(pctx *proc.Context, adaptor proc.DataAdaptor) *Builder {
	return &Builder{
		pctx:       pctx,
		adaptor:    adaptor,
		schedulers: make(map[dag.Source]proc.Scheduler),
	}
}

type Reader struct {
	Layout  order.Layout
	Readers []zio.Reader
}

var _ dag.Source = (*Reader)(nil)

func (*Reader) Source() {}

func (b *Builder) Build(seq *dag.Sequential) ([]zbuf.Puller, error) {
	if !seq.IsEntry() {
		return nil, errors.New("internal error: DAG entry point is not a data source")
	}
	return b.compile(seq, nil)
}

func (b *Builder) Meters() []zbuf.Meter {
	var meters []zbuf.Meter
	for _, sched := range b.schedulers {
		meters = append(meters, sched)
	}
	return meters
}

func (b *Builder) compileLeaf(op dag.Op, parent zbuf.Puller) (zbuf.Puller, error) {
	switch v := op.(type) {
	case *dag.Summarize:
		return compileGroupBy(b.pctx, parent, v)
	case *dag.Cut:
		assignments, err := compileAssignments(v.Args, b.pctx.Zctx)
		if err != nil {
			return nil, err
		}
		lhs, rhs := splitAssignments(assignments)
		cutter, err := expr.NewCutter(b.pctx.Zctx, lhs, rhs)
		if err != nil {
			return nil, err
		}
		if v.Quiet {
			cutter.Quiet()
		}
		return proc.NewApplier(b.pctx, parent, cutter), nil
	case *dag.Drop:
		if len(v.Args) == 0 {
			return nil, errors.New("drop: no fields given")
		}
		fields := make(field.List, 0, len(v.Args))
		for _, e := range v.Args {
			field, ok := e.(*dag.Path)
			if !ok {
				return nil, errors.New("drop: arg not a field")
			}
			fields = append(fields, field.Name)
		}
		dropper := expr.NewDropper(b.pctx.Zctx, fields)
		return proc.NewApplier(b.pctx, parent, dropper), nil
	case *dag.Sort:
		fields, err := CompileExprs(b.pctx.Zctx, v.Args)
		if err != nil {
			return nil, err
		}
		sort, err := sort.New(b.pctx, parent, fields, v.Order, v.NullsFirst)
		if err != nil {
			return nil, fmt.Errorf("compiling sort: %w", err)
		}
		return sort, nil
	case *dag.Head:
		limit := v.Count
		if limit == 0 {
			limit = 1
		}
		return head.New(parent, limit), nil
	case *dag.Tail:
		limit := v.Count
		if limit == 0 {
			limit = 1
		}
		return tail.New(parent, limit), nil
	case *dag.Uniq:
		return uniq.New(b.pctx, parent, v.Cflag), nil
	case *dag.Pass:
		return pass.New(parent), nil
	case *dag.Filter:
		f, err := compileFilter(b.pctx.Zctx, v.Expr)
		if err != nil {
			return nil, fmt.Errorf("compiling filter: %w", err)
		}
		return proc.NewApplier(b.pctx, parent, expr.NewFilterApplier(b.pctx.Zctx, f)), nil
	case *dag.Top:
		fields, err := CompileExprs(b.pctx.Zctx, v.Args)
		if err != nil {
			return nil, fmt.Errorf("compiling top: %w", err)
		}
		return top.New(parent, b.pctx.Zctx, v.Limit, fields, v.Flush), nil
	case *dag.Put:
		clauses, err := compileAssignments(v.Args, b.pctx.Zctx)
		if err != nil {
			return nil, err
		}
		put, err := put.New(b.pctx, parent, clauses)
		if err != nil {
			return nil, err
		}
		return put, nil
	case *dag.Rename:
		var srcs, dsts field.List
		for _, fa := range v.Args {
			dst, err := compileLval(fa.LHS)
			if err != nil {
				return nil, err
			}
			// We call CompileLval on the RHS because renames are
			// restricted to dotted field name expressions.
			src, err := compileLval(fa.RHS)
			if err != nil {
				return nil, err
			}
			if len(dst) != len(src) {
				return nil, fmt.Errorf("cannot rename %s to %s", src, dst)
			}
			// Check that the prefixes match and, if not, report first place
			// that they don't.
			for i := 0; i <= len(src)-2; i++ {
				if src[i] != dst[i] {
					return nil, fmt.Errorf("cannot rename %s to %s (differ in %s vs %s)", src, dst, src[i], dst[i])
				}
			}
			dsts = append(dsts, dst)
			srcs = append(srcs, src)
		}
		renamer := expr.NewRenamer(b.pctx.Zctx, srcs, dsts)
		return proc.NewApplier(b.pctx, parent, renamer), nil
	case *dag.Fuse:
		return fuse.New(b.pctx, parent)
	case *dag.Shape:
		return shape.New(b.pctx, parent)
	case *dag.Join:
		return nil, ErrJoinParents
	case *dag.Explode:
		typ, err := zson.ParseType(b.pctx.Zctx, v.Type)
		if err != nil {
			return nil, err
		}
		args, err := compileExprs(b.pctx.Zctx, v.Args)
		if err != nil {
			return nil, err
		}
		as, err := compileLval(v.As)
		if len(as) != 1 {
			return nil, errors.New("explode field must be a top-level field")
		}
		return explode.New(b.pctx.Zctx, parent, args, typ, as.Leaf())
	case *dag.Over:
		return b.compileOver(parent, v, nil, nil)
	case *dag.Yield:
		exprs, err := compileExprs(b.pctx.Zctx, v.Exprs)
		if err != nil {
			return nil, err
		}
		t := yield.New(parent, exprs)
		return t, nil
	case *dag.Let:
		if v.Over == nil {
			return nil, errors.New("let operator missing over expression in DAG")
		}
		exprs := make([]expr.Evaluator, 0, len(v.Defs))
		names := make([]string, 0, len(v.Defs))
		for _, def := range v.Defs {
			e, err := compileExpr(b.pctx.Zctx, def.Expr)
			if err != nil {
				return nil, err
			}
			exprs = append(exprs, e)
			names = append(names, def.Name)
		}
		return b.compileOver(parent, v.Over, names, exprs)
	default:
		return nil, fmt.Errorf("unknown DAG operator type: %v", v)
	}
}

func (b *Builder) compileOver(parent zbuf.Puller, over *dag.Over, names []string, lets []expr.Evaluator) (zbuf.Puller, error) {
	exprs, err := compileExprs(b.pctx.Zctx, over.Exprs)
	if err != nil {
		return nil, err
	}
	enter := traverse.NewOver(b.pctx, parent, exprs)
	if over.Scope == nil {
		return enter, nil
	}
	scope := enter.AddScope(b.pctx.Context, names, lets)
	exits, err := b.compile(over.Scope, []zbuf.Puller{scope})
	if err != nil {
		return nil, err
	}
	var exit zbuf.Puller
	if len(exits) == 1 {
		exit = exits[0]
	} else {
		// This can happen when output of over body
		// is a split or switch.
		exit = combine.New(b.pctx, exits)
	}
	return scope.NewExit(exit), nil
}

func compileAssignments(assignments []dag.Assignment, zctx *zed.Context) ([]expr.Assignment, error) {
	keys := make([]expr.Assignment, 0, len(assignments))
	for _, assignment := range assignments {
		a, err := CompileAssignment(zctx, &assignment)
		if err != nil {
			return nil, err
		}
		keys = append(keys, a)
	}
	return keys, nil
}

func splitAssignments(assignments []expr.Assignment) (field.List, []expr.Evaluator) {
	n := len(assignments)
	lhs := make(field.List, 0, n)
	rhs := make([]expr.Evaluator, 0, n)
	for _, a := range assignments {
		lhs = append(lhs, a.LHS)
		rhs = append(rhs, a.RHS)
	}
	return lhs, rhs
}

func (b *Builder) compileSequential(seq *dag.Sequential, parents []zbuf.Puller) ([]zbuf.Puller, error) {
	for _, op := range seq.Ops {
		var err error
		parents, err = b.compile(op, parents)
		if err != nil {
			return nil, err
		}
	}
	return parents, nil
}

func (b *Builder) compileParallel(parallel *dag.Parallel, parents []zbuf.Puller) ([]zbuf.Puller, error) {
	if len(parents) == 0 {
		var procs []zbuf.Puller
		for _, op := range parallel.Ops {
			proc, err := b.compile(op, nil)
			if err != nil {
				return nil, err
			}
			procs = append(procs, proc...)
		}
		return procs, nil
	}
	n := len(parallel.Ops)
	if len(parents) == 1 {
		// Single parent: insert a splitter for n-way fanout.
		parents = split.New(b.pctx, parents[0], n)
	}
	if len(parents) != n {
		return nil, fmt.Errorf("parallel input mismatch: %d parents with %d flowgraph paths", len(parents), len(parallel.Ops))
	}
	var procs []zbuf.Puller
	for k := 0; k < n; k++ {
		proc, err := b.compile(parallel.Ops[k], []zbuf.Puller{parents[k]})
		if err != nil {
			return nil, err
		}
		procs = append(procs, proc...)
	}
	return procs, nil
}

func (b *Builder) compileExprSwitch(swtch *dag.Switch, parents []zbuf.Puller) ([]zbuf.Puller, error) {
	if len(parents) != 1 {
		return nil, errors.New("expression switch has multiple parents")
	}
	e, err := compileExpr(b.pctx.Zctx, swtch.Expr)
	if err != nil {
		return nil, err
	}
	s := exprswitch.New(b.pctx, parents[0], e)
	var exits []zbuf.Puller
	for _, c := range swtch.Cases {
		var val *zed.Value
		if c.Expr != nil {
			val, err = EvalAtCompileTime(b.pctx.Zctx, c.Expr)
			if err != nil {
				return nil, err
			}
		}
		parents, err := b.compile(c.Op, []zbuf.Puller{s.AddCase(val)})
		if err != nil {
			return nil, err
		}
		exits = append(exits, parents...)
	}
	return exits, nil
}

func (b *Builder) compileSwitch(swtch *dag.Switch, parents []zbuf.Puller) ([]zbuf.Puller, error) {
	n := len(swtch.Cases)
	if len(parents) == 1 {
		// Single parent: insert a switcher and wire to each branch.
		switcher := switcher.New(b.pctx, parents[0])
		parents = []zbuf.Puller{}
		for _, c := range swtch.Cases {
			f, err := compileFilter(b.pctx.Zctx, c.Expr)
			if err != nil {
				return nil, fmt.Errorf("compiling switch case filter: %w", err)
			}
			sc := switcher.AddCase(f)
			parents = append(parents, sc)
		}
	}
	if len(parents) != n {
		return nil, fmt.Errorf("proc.compileSwitch: %d parents for switch proc with %d branches", len(parents), len(swtch.Cases))
	}
	var procs []zbuf.Puller
	for k := 0; k < n; k++ {
		proc, err := b.compile(swtch.Cases[k].Op, []zbuf.Puller{parents[k]})
		if err != nil {
			return nil, err
		}
		procs = append(procs, proc...)
	}
	return procs, nil
}

// compile compiles a DAG into a graph of runtime operators, and returns
// the leaves.
func (b *Builder) compile(op dag.Op, parents []zbuf.Puller) ([]zbuf.Puller, error) {
	switch op := op.(type) {
	case *dag.Sequential:
		if len(op.Ops) == 0 {
			return nil, errors.New("sequential proc without procs")
		}
		return b.compileSequential(op, parents)
	case *dag.Parallel:
		return b.compileParallel(op, parents)
	case *dag.Switch:
		if op.Expr != nil {
			return b.compileExprSwitch(op, parents)
		}
		return b.compileSwitch(op, parents)
	case *dag.From:
		if len(parents) > 1 {
			return nil, errors.New("'from' operator can have at most one parent")
		}
		var parent zbuf.Puller
		if len(parents) == 1 {
			parent = parents[0]
		}
		return b.compileFrom(op, parent)
	case *dag.Join:
		if len(parents) != 2 {
			return nil, ErrJoinParents
		}
		assignments, err := compileAssignments(op.Args, b.pctx.Zctx)
		if err != nil {
			return nil, err
		}
		lhs, rhs := splitAssignments(assignments)
		leftKey, err := compileExpr(b.pctx.Zctx, op.LeftKey)
		if err != nil {
			return nil, err
		}
		rightKey, err := compileExpr(b.pctx.Zctx, op.RightKey)
		if err != nil {
			return nil, err
		}
		leftParent, rightParent := parents[0], parents[1]
		var anti, inner bool
		switch op.Style {
		case "anti":
			anti = true
		case "inner":
			inner = true
		case "left":
		case "right":
			leftKey, rightKey = rightKey, leftKey
			leftParent, rightParent = rightParent, leftParent
		default:
			return nil, fmt.Errorf("unknown kind of join: '%s'", op.Style)
		}
		join, err := join.New(b.pctx, anti, inner, leftParent, rightParent, leftKey, rightKey, lhs, rhs)
		if err != nil {
			return nil, err
		}
		return []zbuf.Puller{join}, nil
	case *dag.Merge:
		layout := order.NewLayout(op.Order, field.List{op.Key})
		cmp := zbuf.NewCompareFn(b.pctx.Zctx, layout)
		return []zbuf.Puller{merge.New(b.pctx, parents, cmp)}, nil
	default:
		var parent zbuf.Puller
		if len(parents) == 1 {
			parent = parents[0]
		} else {
			parent = combine.New(b.pctx, parents)
		}
		p, err := b.compileLeaf(op, parent)
		if err != nil {
			return nil, err
		}
		return []zbuf.Puller{p}, nil
	}
}

func (b *Builder) compileFrom(from *dag.From, parent zbuf.Puller) ([]zbuf.Puller, error) {
	var parents []zbuf.Puller
	var npass int
	for k := range from.Trunks {
		outputs, err := b.compileTrunk(&from.Trunks[k], parent)
		if err != nil {
			return nil, err
		}
		if _, ok := from.Trunks[k].Source.(*dag.Pass); ok {
			npass++
		}
		parents = append(parents, outputs...)
	}
	if parent == nil && npass > 0 {
		return nil, errors.New("no data source for 'from operator' pass-through branch")
	}
	if parent != nil {
		if npass > 1 {
			return nil, errors.New("cannot have multiple pass-through branches in 'from operator'")
		}
		if npass == 0 {
			return nil, errors.New("upstream data source blocked by 'from operator'")
		}
	}
	return parents, nil
}

func (b *Builder) compileTrunk(trunk *dag.Trunk, parent zbuf.Puller) ([]zbuf.Puller, error) {
	pushdown, err := b.PushdownOf(trunk)
	if err != nil {
		return nil, err
	}
	var source zbuf.Puller
	switch src := trunk.Source.(type) {
	case *Reader:
		sched := &readerScheduler{
			ctx:     b.pctx.Context,
			filter:  pushdown,
			readers: src.Readers,
		}
		source = from.NewScheduler(b.pctx, sched)
		b.schedulers[src] = sched
	case *dag.Pass:
		source = parent
	case *dag.Pool:
		// We keep a map of schedulers indexed by *dag.Pool so we
		// properly share parallel instances of a given scheduler
		// across different DAG entry points.  The scanners from a
		// common lake.ScanScheduler are distributed across the collection
		// of proc.From operators.
		sched, ok := b.schedulers[src]
		if !ok {
			span, err := b.compileRange(src, src.ScanLower, src.ScanUpper)
			if err != nil {
				return nil, err
			}
			sched, err = b.adaptor.NewScheduler(b.pctx.Context, b.pctx.Zctx, src, span, pushdown, trunk.Pushdown.Index)
			if err != nil {
				return nil, err
			}
			b.schedulers[src] = sched
		}
		source = from.NewScheduler(b.pctx, sched)
	case *dag.PoolMeta:
		sched, ok := b.schedulers[src]
		if !ok {
			sched, err = b.adaptor.NewScheduler(b.pctx.Context, b.pctx.Zctx, src, nil, pushdown, nil)
			if err != nil {
				return nil, err
			}
			b.schedulers[src] = sched
		}
		source = from.NewScheduler(b.pctx, sched)
	case *dag.CommitMeta:
		sched, ok := b.schedulers[src]
		if !ok {
			span, err := b.compileRange(src, src.ScanLower, src.ScanUpper)
			if err != nil {
				return nil, err
			}
			sched, err = b.adaptor.NewScheduler(b.pctx.Context, b.pctx.Zctx, src, span, pushdown, nil)
			if err != nil {
				return nil, err
			}
			b.schedulers[src] = sched
		}
		source = from.NewScheduler(b.pctx, sched)
	case *dag.LakeMeta:
		sched, ok := b.schedulers[src]
		if !ok {
			sched, err = b.adaptor.NewScheduler(b.pctx.Context, b.pctx.Zctx, src, nil, pushdown, nil)
			if err != nil {
				return nil, err
			}
			b.schedulers[src] = sched
		}
		source = from.NewScheduler(b.pctx, sched)
	case *dag.HTTP:
		puller, err := b.adaptor.Open(b.pctx.Context, b.pctx.Zctx, src.URL, pushdown)
		if err != nil {
			return nil, err
		}
		source = from.NewPuller(b.pctx, puller)
	case *dag.File:
		scanner, err := b.adaptor.Open(b.pctx.Context, b.pctx.Zctx, src.Path, pushdown)
		if err != nil {
			return nil, err
		}
		source = from.NewPuller(b.pctx, scanner)
	default:
		return nil, fmt.Errorf("Builder.compileTrunk: unknown type: %T", src)
	}
	if trunk.Seq == nil {
		return []zbuf.Puller{source}, nil
	}
	return b.compileSequential(trunk.Seq, []zbuf.Puller{source})
}

func (b *Builder) compileRange(src dag.Source, exprLower, exprUpper dag.Expr) (extent.Span, error) {
	lower := &zed.Value{zed.TypeNull, nil}
	upper := &zed.Value{zed.TypeNull, nil}
	if exprLower != nil {
		var err error
		lower, err = EvalAtCompileTime(b.pctx.Zctx, exprLower)
		if err != nil {
			return nil, err
		}
	}
	if exprUpper != nil {
		var err error
		upper, err = EvalAtCompileTime(b.pctx.Zctx, exprUpper)
		if err != nil {
			return nil, err
		}
	}
	var span extent.Span
	if lower.Bytes != nil || upper.Bytes != nil {
		layout := b.adaptor.Layout(b.pctx.Context, src)
		span = extent.NewGenericFromOrder(*lower, *upper, layout.Order)
	}
	return span, nil
}

func (b *Builder) PushdownOf(trunk *dag.Trunk) (*Filter, error) {
	var filter *Filter
	if trunk.Pushdown.Scan != nil {
		f, ok := trunk.Pushdown.Scan.(*dag.Filter)
		if !ok {
			return nil, errors.New("non-filter pushdown operator not yet supported")
		}
		filter = &Filter{b, f.Expr}
	}
	return filter, nil
}

func EvalAtCompileTime(zctx *zed.Context, in dag.Expr) (val *zed.Value, err error) {
	if in == nil {
		return zed.Null, nil
	}
	e, err := compileExpr(zctx, in)
	if err != nil {
		return nil, err
	}
	// Catch panic as the runtime will panic if there is a
	// reference to a var not in scope, a field access null this, etc.
	defer func() {
		if recover() != nil {
			err = errors.New("panic")
		}
	}()
	return e.Eval(expr.NewContext(), zed.Null), nil
}

type readerScheduler struct {
	ctx      context.Context
	filter   *Filter
	readers  []zio.Reader
	scanner  zbuf.Scanner
	progress zbuf.Progress
}

func (r *readerScheduler) PullScanTask() (zbuf.PullerCloser, error) {
	if r.scanner != nil {
		r.progress.Add(r.scanner.Progress())
		r.scanner = nil
	}
	if len(r.readers) == 0 {
		return nil, nil
	}
	zr := r.readers[0]
	r.readers = r.readers[1:]
	s, err := zbuf.NewScanner(r.ctx, zr, r.filter)
	if err != nil {
		return nil, err
	}
	r.scanner = s
	if stringer, ok := zr.(fmt.Stringer); ok {
		s = zbuf.NamedScanner(s, stringer.String())
	}
	return &donePullerCloser{zbuf.ScannerNopCloser(s), r}, nil
}

func (r *readerScheduler) Progress() zbuf.Progress {
	// Add the cumulative progress to the current scanner's progress.
	progress := r.progress
	if r.scanner != nil {
		progress.Add(r.scanner.Progress())
	}
	return progress
}

type donePullerCloser struct {
	zbuf.PullerCloser
	sched *readerScheduler
}

func (d *donePullerCloser) Pull(done bool) (zbuf.Batch, error) {
	if done {
		d.sched.readers = nil
		return nil, nil
	}
	return d.PullerCloser.Pull(false)
}

package kernel

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/data"
	"github.com/brimdata/zed/compiler/optimizer/demand"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/runtime/op/combine"
	"github.com/brimdata/zed/runtime/op/explode"
	"github.com/brimdata/zed/runtime/op/exprswitch"
	"github.com/brimdata/zed/runtime/op/fork"
	"github.com/brimdata/zed/runtime/op/fuse"
	"github.com/brimdata/zed/runtime/op/head"
	"github.com/brimdata/zed/runtime/op/join"
	"github.com/brimdata/zed/runtime/op/load"
	"github.com/brimdata/zed/runtime/op/merge"
	"github.com/brimdata/zed/runtime/op/meta"
	"github.com/brimdata/zed/runtime/op/pass"
	"github.com/brimdata/zed/runtime/op/shape"
	"github.com/brimdata/zed/runtime/op/sort"
	"github.com/brimdata/zed/runtime/op/switcher"
	"github.com/brimdata/zed/runtime/op/tail"
	"github.com/brimdata/zed/runtime/op/top"
	"github.com/brimdata/zed/runtime/op/traverse"
	"github.com/brimdata/zed/runtime/op/uniq"
	"github.com/brimdata/zed/runtime/op/yield"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

var ErrJoinParents = errors.New("join requires two upstream parallel query paths")

type Builder struct {
	octx     *op.Context
	mctx     *zed.Context
	source   *data.Source
	readers  []zio.Reader
	progress *zbuf.Progress
	deletes  *sync.Map
	funcs    map[string]expr.Function
}

func NewBuilder(octx *op.Context, source *data.Source) *Builder {
	return &Builder{
		octx:   octx,
		mctx:   zed.NewContext(),
		source: source,
		progress: &zbuf.Progress{
			BytesRead:      0,
			BytesMatched:   0,
			RecordsRead:    0,
			RecordsMatched: 0,
		},
		funcs: make(map[string]expr.Function),
	}
}

// Build builds a flowgraph for seq.  If seq contains a dag.DefaultSource, it
// will read from readers.
func (b *Builder) Build(seq dag.Seq, readers ...zio.Reader) ([]zbuf.Puller, error) {
	if !isEntry(seq) {
		return nil, errors.New("internal error: DAG entry point is not a data source")
	}
	b.readers = readers
	return b.compileSeq(seq, nil)
}

func (b *Builder) zctx() *zed.Context {
	return b.octx.Zctx
}

func (b *Builder) Meter() zbuf.Meter {
	return b.progress
}

func (b *Builder) Deletes() *sync.Map {
	return b.deletes
}

func (b *Builder) compileLeaf(o dag.Op, parent zbuf.Puller) (zbuf.Puller, error) {
	switch v := o.(type) {
	case *dag.Summarize:
		return b.compileGroupBy(parent, v)
	case *dag.Cut:
		assignments, err := b.compileAssignments(v.Args)
		if err != nil {
			return nil, err
		}
		lhs, rhs := splitAssignments(assignments)
		cutter := expr.NewCutter(b.octx.Zctx, lhs, rhs)
		if v.Quiet {
			cutter.Quiet()
		}
		return op.NewApplier(b.octx, parent, cutter), nil
	case *dag.Drop:
		if len(v.Args) == 0 {
			return nil, errors.New("drop: no fields given")
		}
		fields := make(field.List, 0, len(v.Args))
		for _, e := range v.Args {
			field, ok := e.(*dag.This)
			if !ok {
				return nil, errors.New("drop: arg not a field")
			}
			fields = append(fields, field.Path)
		}
		dropper := expr.NewDropper(b.octx.Zctx, fields)
		return op.NewApplier(b.octx, parent, dropper), nil
	case *dag.Sort:
		fields, err := b.compileExprs(v.Args)
		if err != nil {
			return nil, err
		}
		sort, err := sort.New(b.octx, parent, fields, v.Order, v.NullsFirst)
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
		return uniq.New(b.octx, parent, v.Cflag), nil
	case *dag.Pass:
		return pass.New(parent), nil
	case *dag.Filter:
		f, err := b.compileExpr(v.Expr)
		if err != nil {
			return nil, fmt.Errorf("compiling filter: %w", err)
		}
		return op.NewApplier(b.octx, parent, expr.NewFilterApplier(b.octx.Zctx, f)), nil
	case *dag.Top:
		fields, err := b.compileExprs(v.Args)
		if err != nil {
			return nil, fmt.Errorf("compiling top: %w", err)
		}
		return top.New(b.octx.Zctx, parent, v.Limit, fields, v.Flush), nil
	case *dag.Put:
		clauses, err := b.compileAssignments(v.Args)
		if err != nil {
			return nil, err
		}
		putter := expr.NewPutter(b.octx.Zctx, clauses)
		return op.NewApplier(b.octx, parent, putter), nil
	case *dag.Rename:
		var srcs, dsts []*expr.Lval
		for _, a := range v.Args {
			src, err := b.compileLval(a.RHS)
			if err != nil {
				return nil, err
			}
			dst, err := b.compileLval(a.LHS)
			if err != nil {
				return nil, err
			}
			srcs = append(srcs, src)
			dsts = append(dsts, dst)
		}
		renamer := expr.NewRenamer(b.octx.Zctx, srcs, dsts)
		return op.NewApplier(b.octx, parent, renamer), nil
	case *dag.Fuse:
		return fuse.New(b.octx, parent)
	case *dag.Shape:
		return shape.New(b.octx, parent)
	case *dag.Join:
		return nil, ErrJoinParents
	case *dag.Merge:
		return nil, errors.New("merge: multiple upstream paths required")
	case *dag.Explode:
		typ, err := zson.ParseType(b.octx.Zctx, v.Type)
		if err != nil {
			return nil, err
		}
		args, err := b.compileExprs(v.Args)
		if err != nil {
			return nil, err
		}
		return explode.New(b.octx.Zctx, parent, args, typ, v.As)
	case *dag.Over:
		return b.compileOver(parent, v)
	case *dag.Yield:
		exprs, err := b.compileExprs(v.Exprs)
		if err != nil {
			return nil, err
		}
		t := yield.New(parent, exprs)
		return t, nil
	case *dag.PoolScan:
		if parent != nil {
			return nil, errors.New("internal error: pool scan cannot have a parent operator")
		}
		return b.compilePoolScan(v)
	case *dag.PoolMetaScan:
		return meta.NewPoolMetaScanner(b.octx.Context, b.octx.Zctx, b.source.Lake(), v.ID, v.Meta)
	case *dag.CommitMetaScan:
		var pruner expr.Evaluator
		if v.Tap && v.KeyPruner != nil {
			var err error
			pruner, err = compileExpr(v.KeyPruner)
			if err != nil {
				return nil, err
			}
		}
		return meta.NewCommitMetaScanner(b.octx.Context, b.octx.Zctx, b.source.Lake(), v.Pool, v.Commit, v.Meta, pruner)
	case *dag.LakeMetaScan:
		return meta.NewLakeMetaScanner(b.octx.Context, b.octx.Zctx, b.source.Lake(), v.Meta)
	case *dag.HTTPScan:
		body := strings.NewReader(v.Body)
		return b.source.OpenHTTP(b.octx.Context, b.octx.Zctx, v.URL, v.Format, v.Method, v.Headers, body, demand.All())
	case *dag.FileScan:
		return b.source.Open(b.octx.Context, b.octx.Zctx, v.Path, v.Format, b.PushdownOf(v.Filter), demand.All())
	case *dag.DefaultScan:
		pushdown := b.PushdownOf(v.Filter)
		if len(b.readers) == 1 {
			return zbuf.NewScanner(b.octx.Context, b.readers[0], pushdown)
		}
		scanners := make([]zbuf.Scanner, 0, len(b.readers))
		for _, r := range b.readers {
			scanner, err := zbuf.NewScanner(b.octx.Context, r, pushdown)
			if err != nil {
				return nil, err
			}
			scanners = append(scanners, scanner)
		}
		return zbuf.MultiScanner(scanners...), nil
	case *dag.Lister:
		if parent != nil {
			return nil, errors.New("internal error: data source cannot have a parent operator")
		}
		pool, err := b.lookupPool(v.Pool)
		if err != nil {
			return nil, err
		}
		var pruner expr.Evaluator
		if v.KeyPruner != nil {
			pruner, err = compileExpr(v.KeyPruner)
			if err != nil {
				return nil, err
			}
		}
		return meta.NewSortedLister(b.octx.Context, b.mctx, b.source.Lake(), pool, v.Commit, pruner)
	case *dag.Slicer:
		return meta.NewSlicer(parent, b.mctx), nil
	case *dag.SeqScan:
		pool, err := b.lookupPool(v.Pool)
		if err != nil {
			return nil, err
		}
		var pruner expr.Evaluator
		if v.KeyPruner != nil {
			pruner, err = compileExpr(v.KeyPruner)
			if err != nil {
				return nil, err
			}
		}
		return meta.NewSequenceScanner(b.octx, parent, pool, b.PushdownOf(v.Filter), pruner, b.progress), nil
	case *dag.Deleter:
		pool, err := b.lookupPool(v.Pool)
		if err != nil {
			return nil, err
		}
		var pruner expr.Evaluator
		if v.KeyPruner != nil {
			pruner, err = compileExpr(v.KeyPruner)
			if err != nil {
				return nil, err
			}
		}
		if b.deletes == nil {
			b.deletes = &sync.Map{}
		}
		var filter *DeleteFilter
		if f := b.PushdownOf(v.Where); f != nil {
			filter = &DeleteFilter{f}
		}
		return meta.NewDeleter(b.octx, parent, pool, filter, pruner, b.progress, b.deletes), nil
	case *dag.Load:
		return load.New(b.octx, b.source.Lake(), parent, v.Pool, v.Branch, v.Author, v.Message, v.Meta), nil
	default:
		return nil, fmt.Errorf("unknown DAG operator type: %v", v)
	}
}

func (b *Builder) compileDefs(defs []dag.Def) ([]string, []expr.Evaluator, error) {
	exprs := make([]expr.Evaluator, 0, len(defs))
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		e, err := b.compileExpr(def.Expr)
		if err != nil {
			return nil, nil, err
		}
		exprs = append(exprs, e)
		names = append(names, def.Name)
	}
	return names, exprs, nil
}

func (b *Builder) compileOver(parent zbuf.Puller, over *dag.Over) (zbuf.Puller, error) {
	if len(over.Defs) != 0 && over.Body == nil {
		return nil, errors.New("internal error: over operator has defs but no body")
	}
	withNames, withExprs, err := b.compileDefs(over.Defs)
	if err != nil {
		return nil, err
	}
	exprs, err := b.compileExprs(over.Exprs)
	if err != nil {
		return nil, err
	}
	enter := traverse.NewOver(b.octx, parent, exprs)
	if over.Body == nil {
		return enter, nil
	}
	scope := enter.AddScope(b.octx.Context, withNames, withExprs)
	exits, err := b.compileSeq(over.Body, []zbuf.Puller{scope})
	if err != nil {
		return nil, err
	}
	var exit zbuf.Puller
	if len(exits) == 1 {
		exit = exits[0]
	} else {
		// This can happen when output of over body
		// is a fork or switch.
		exit = combine.New(b.octx, exits)
	}
	return scope.NewExit(exit), nil
}

func (b *Builder) compileAssignments(assignments []dag.Assignment) ([]expr.Assignment, error) {
	keys := make([]expr.Assignment, 0, len(assignments))
	for _, assignment := range assignments {
		a, err := b.compileAssignment(&assignment)
		if err != nil {
			return nil, err
		}
		keys = append(keys, a)
	}
	return keys, nil
}

func splitAssignments(assignments []expr.Assignment) ([]*expr.Lval, []expr.Evaluator) {
	n := len(assignments)
	lhs := make([]*expr.Lval, 0, n)
	rhs := make([]expr.Evaluator, 0, n)
	for _, a := range assignments {
		lhs = append(lhs, a.LHS)
		rhs = append(rhs, a.RHS)
	}
	return lhs, rhs
}

func (b *Builder) compileSeq(seq dag.Seq, parents []zbuf.Puller) ([]zbuf.Puller, error) {
	for _, o := range seq {
		var err error
		parents, err = b.compile(o, parents)
		if err != nil {
			return nil, err
		}
	}
	return parents, nil
}

func (b *Builder) compileScope(scope *dag.Scope, parents []zbuf.Puller) ([]zbuf.Puller, error) {
	if err := b.compileFuncs(scope.Funcs); err != nil {
		return nil, err
	}
	return b.compileSeq(scope.Body, parents)
}

func (b *Builder) compileFork(par *dag.Fork, parents []zbuf.Puller) ([]zbuf.Puller, error) {
	var f *fork.Op
	switch len(parents) {
	case 0:
		// No parents: no need for a fork since every op gets a nil parent.
	case 1:
		// Single parent: insert a fork for n-way fanout.
		f = fork.New(b.octx, parents[0])
	default:
		// Multiple parents: insert a combine followed by a fork for n-way fanout.
		f = fork.New(b.octx, combine.New(b.octx, parents))
	}
	var ops []zbuf.Puller
	for _, seq := range par.Paths {
		var parent zbuf.Puller
		if f != nil && !isEntry(seq) {
			parent = f.AddExit()
		}
		op, err := b.compileSeq(seq, []zbuf.Puller{parent})
		if err != nil {
			return nil, err
		}
		ops = append(ops, op...)
	}
	return ops, nil
}

func (b *Builder) compileScatter(par *dag.Scatter, parents []zbuf.Puller) ([]zbuf.Puller, error) {
	if len(parents) != 1 {
		return nil, errors.New("internal error: scatter operator requires a single parent")
	}
	var ops []zbuf.Puller
	for _, o := range par.Paths {
		op, err := b.compileSeq(o, parents[:1])
		if err != nil {
			return nil, err
		}
		ops = append(ops, op...)
	}
	return ops, nil
}

func (b *Builder) compileFuncs(fns []*dag.Func) error {
	udfs := make([]*expr.UDF, 0, len(fns))
	for _, f := range fns {
		if _, ok := b.funcs[f.Name]; ok {
			return fmt.Errorf("internal error: func %q declared twice", f.Name)
		}
		u := &expr.UDF{}
		b.funcs[f.Name] = u
		udfs = append(udfs, u)
	}
	for i := range fns {
		var err error
		if udfs[i].Body, err = b.compileExpr(fns[i].Expr); err != nil {
			return err
		}
	}
	return nil
}

func (b *Builder) compileExprSwitch(swtch *dag.Switch, parents []zbuf.Puller) ([]zbuf.Puller, error) {
	parent := parents[0]
	if len(parents) > 1 {
		parent = combine.New(b.octx, parents)
	}
	e, err := b.compileExpr(swtch.Expr)
	if err != nil {
		return nil, err
	}
	s := exprswitch.New(b.octx, parent, e)
	var exits []zbuf.Puller
	for _, c := range swtch.Cases {
		var val *zed.Value
		if c.Expr != nil {
			val, err = b.evalAtCompileTime(c.Expr)
			if err != nil {
				return nil, err
			}
			if val.IsError() {
				return nil, errors.New("switch case is not a constant expression")
			}
		}
		parents, err := b.compileSeq(c.Path, []zbuf.Puller{s.AddCase(val)})
		if err != nil {
			return nil, err
		}
		exits = append(exits, parents...)
	}
	return exits, nil
}

func (b *Builder) compileSwitch(swtch *dag.Switch, parents []zbuf.Puller) ([]zbuf.Puller, error) {
	parent := parents[0]
	if len(parents) > 1 {
		parent = combine.New(b.octx, parents)
	}
	n := len(swtch.Cases)
	switcher := switcher.New(b.octx, parent)
	parents = []zbuf.Puller{}
	for _, c := range swtch.Cases {
		f, err := b.compileExpr(c.Expr)
		if err != nil {
			return nil, fmt.Errorf("compiling switch case filter: %w", err)
		}
		sc := switcher.AddCase(f)
		parents = append(parents, sc)
	}
	var ops []zbuf.Puller
	for k := 0; k < n; k++ {
		o, err := b.compileSeq(swtch.Cases[k].Path, []zbuf.Puller{parents[k]})
		if err != nil {
			return nil, err
		}
		ops = append(ops, o...)
	}
	return ops, nil
}

// compile compiles a DAG into a graph of runtime operators, and returns
// the leaves.
func (b *Builder) compile(o dag.Op, parents []zbuf.Puller) ([]zbuf.Puller, error) {
	switch o := o.(type) {
	case *dag.Fork:
		return b.compileFork(o, parents)
	case *dag.Scatter:
		return b.compileScatter(o, parents)
	case *dag.Scope:
		return b.compileScope(o, parents)
	case *dag.Switch:
		if o.Expr != nil {
			return b.compileExprSwitch(o, parents)
		}
		return b.compileSwitch(o, parents)
	case *dag.Join:
		if len(parents) != 2 {
			return nil, ErrJoinParents
		}
		assignments, err := b.compileAssignments(o.Args)
		if err != nil {
			return nil, err
		}
		lhs, rhs := splitAssignments(assignments)
		leftKey, err := b.compileExpr(o.LeftKey)
		if err != nil {
			return nil, err
		}
		rightKey, err := b.compileExpr(o.RightKey)
		if err != nil {
			return nil, err
		}
		leftParent, rightParent := parents[0], parents[1]
		leftDir, rightDir := o.LeftDir, o.RightDir
		var anti, inner bool
		switch o.Style {
		case "anti":
			anti = true
		case "inner":
			inner = true
		case "left":
		case "right":
			leftKey, rightKey = rightKey, leftKey
			leftParent, rightParent = rightParent, leftParent
			leftDir, rightDir = rightDir, leftDir
		default:
			return nil, fmt.Errorf("unknown kind of join: '%s'", o.Style)
		}
		join, err := join.New(b.octx, anti, inner, leftParent, rightParent, leftKey, rightKey, leftDir, rightDir, lhs, rhs)
		if err != nil {
			return nil, err
		}
		return []zbuf.Puller{join}, nil
	case *dag.Merge:
		e, err := b.compileExpr(o.Expr)
		if err != nil {
			return nil, err
		}
		cmp := expr.NewComparator(true, o.Order == order.Desc, e).WithMissingAsNull()
		return []zbuf.Puller{merge.New(b.octx, parents, cmp.Compare)}, nil
	case *dag.Combine:
		return []zbuf.Puller{combine.New(b.octx, parents)}, nil
	default:
		var parent zbuf.Puller
		if len(parents) == 1 {
			parent = parents[0]
		} else if len(parents) > 1 {
			parent = combine.New(b.octx, parents)
		}
		p, err := b.compileLeaf(o, parent)
		if err != nil {
			return nil, err
		}
		return []zbuf.Puller{p}, nil
	}
}

func (b *Builder) compilePoolScan(scan *dag.PoolScan) (zbuf.Puller, error) {
	// Here we convert PoolScan to lister->slicer->seqscan for the slow path as
	// optimizer should do this conversion, but this allows us to run
	// unoptimized scans too.
	pool, err := b.lookupPool(scan.ID)
	if err != nil {
		return nil, err
	}
	l, err := meta.NewSortedLister(b.octx.Context, b.mctx, b.source.Lake(), pool, scan.Commit, nil)
	if err != nil {
		return nil, err
	}
	slicer := meta.NewSlicer(l, b.mctx)
	return meta.NewSequenceScanner(b.octx, slicer, pool, nil, nil, b.progress), nil
}

func (b *Builder) PushdownOf(e dag.Expr) *Filter {
	if e == nil {
		return nil
	}
	return &Filter{e, b}
}

func (b *Builder) lookupPool(id ksuid.KSUID) (*lake.Pool, error) {
	if b.source == nil || b.source.Lake() == nil {
		return nil, errors.New("internal error: lake operation cannot be used in non-lake context")
	}
	// This is fast because of the pool cache in the lake.
	return b.source.Lake().OpenPool(b.octx.Context, id)
}

func (b *Builder) evalAtCompileTime(in dag.Expr) (val *zed.Value, err error) {
	if in == nil {
		return zed.Null, nil
	}
	e, err := b.compileExpr(in)
	if err != nil {
		return nil, err
	}
	// Catch panic as the runtime will panic if there is a
	// reference to a var not in scope, a field access null this, etc.
	defer func() {
		if recover() != nil {
			val = b.zctx().Missing()
		}
	}()
	return e.Eval(expr.NewContext(), b.zctx().Missing()), nil
}

func compileExpr(in dag.Expr) (expr.Evaluator, error) {
	b := NewBuilder(op.NewContext(context.Background(), zed.NewContext(), nil), nil)
	return b.compileExpr(in)
}

func EvalAtCompileTime(zctx *zed.Context, in dag.Expr) (val *zed.Value, err error) {
	// We pass in a nil adaptor, which causes a panic for anything adaptor
	// related, which is not currently allowed in an expression sub-query.
	b := NewBuilder(op.NewContext(context.Background(), zctx, nil), nil)
	return b.evalAtCompileTime(in)
}

func isEntry(seq dag.Seq) bool {
	if len(seq) == 0 {
		return false
	}
	switch op := seq[0].(type) {
	case *dag.Lister, *dag.DefaultScan, *dag.FileScan, *dag.HTTPScan, *dag.PoolScan, *dag.LakeMetaScan, *dag.PoolMetaScan, *dag.CommitMetaScan:
		return true
	case *dag.Scope:
		return isEntry(op.Body)
	case *dag.Fork:
		return len(op.Paths) > 0 && !slices.ContainsFunc(op.Paths, func(seq dag.Seq) bool {
			return !isEntry(seq)
		})
	}
	return false
}

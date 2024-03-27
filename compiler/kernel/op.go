package kernel

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/data"
	"github.com/brimdata/zed/compiler/optimizer"
	"github.com/brimdata/zed/compiler/optimizer/demand"
	"github.com/brimdata/zed/lake"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/runtime/sam/expr"
	"github.com/brimdata/zed/runtime/sam/op"
	"github.com/brimdata/zed/runtime/sam/op/combine"
	"github.com/brimdata/zed/runtime/sam/op/explode"
	"github.com/brimdata/zed/runtime/sam/op/exprswitch"
	"github.com/brimdata/zed/runtime/sam/op/fork"
	"github.com/brimdata/zed/runtime/sam/op/fuse"
	"github.com/brimdata/zed/runtime/sam/op/head"
	"github.com/brimdata/zed/runtime/sam/op/join"
	"github.com/brimdata/zed/runtime/sam/op/load"
	"github.com/brimdata/zed/runtime/sam/op/merge"
	"github.com/brimdata/zed/runtime/sam/op/meta"
	"github.com/brimdata/zed/runtime/sam/op/pass"
	"github.com/brimdata/zed/runtime/sam/op/shape"
	"github.com/brimdata/zed/runtime/sam/op/sort"
	"github.com/brimdata/zed/runtime/sam/op/switcher"
	"github.com/brimdata/zed/runtime/sam/op/tail"
	"github.com/brimdata/zed/runtime/sam/op/top"
	"github.com/brimdata/zed/runtime/sam/op/traverse"
	"github.com/brimdata/zed/runtime/sam/op/uniq"
	"github.com/brimdata/zed/runtime/sam/op/yield"
	"github.com/brimdata/zed/runtime/vam"
	vop "github.com/brimdata/zed/runtime/vam/op"
	"github.com/brimdata/zed/vector"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

var ErrJoinParents = errors.New("join requires two upstream parallel query paths")

type Builder struct {
	rctx      *runtime.Context
	mctx      *zed.Context
	source    *data.Source
	readers   []zio.Reader
	progress  *zbuf.Progress
	arena     *zed.Arena // For zed.Values created during compilation.
	deletes   *sync.Map
	udfs      map[string]dag.Expr
	udfStack  map[string]*expr.UDF
	resetters expr.Resetters
}

func NewBuilder(rctx *runtime.Context, source *data.Source) *Builder {
	arena := zed.NewArena()
	rctx.KeepAlive(arena)
	return &Builder{
		rctx:   rctx,
		mctx:   zed.NewContext(),
		source: source,
		progress: &zbuf.Progress{
			BytesRead:      0,
			BytesMatched:   0,
			RecordsRead:    0,
			RecordsMatched: 0,
		},
		arena: arena,
		udfs:  make(map[string]dag.Expr),
	}
}

func (b *Builder) clone(arena *zed.Arena) *Builder {
	bb := *b
	bb.arena = arena
	return &bb
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

func (b *Builder) BuildWithPuller(seq dag.Seq, parent vector.Puller) ([]vector.Puller, error) {
	return b.compileVamSeq(seq, []vector.Puller{parent})
}

func (b *Builder) BuildVamToSeqFilter(filter dag.Expr, poolID, commitID ksuid.KSUID) (zbuf.Puller, error) {
	pool, err := b.source.Lake().OpenPool(b.rctx.Context, poolID)
	if err != nil {
		return nil, err
	}
	e, err := b.compileVamExpr(filter)
	if err != nil {
		return nil, err
	}
	l, err := meta.NewSortedLister(b.rctx.Context, b.mctx, pool, commitID, nil)
	if err != nil {
		return nil, err
	}
	cache := b.source.Lake().VectorCache()
	project, _ := optimizer.FieldsOf(filter)
	search, err := vop.NewSearcher(b.rctx, cache, l, pool, e, project)
	if err != nil {
		return nil, err
	}
	return meta.NewSearchScanner(b.rctx, search, pool, b.PushdownOf(filter), b.progress), nil
}

func (b *Builder) zctx() *zed.Context {
	return b.rctx.Zctx
}

func (b *Builder) Meter() zbuf.Meter {
	return b.progress
}

func (b *Builder) Deletes() *sync.Map {
	return b.deletes
}

func (b *Builder) resetResetters() {
	b.resetters = nil
}

func (b *Builder) compileLeaf(o dag.Op, parent zbuf.Puller) (zbuf.Puller, error) {
	switch v := o.(type) {
	case *dag.Summarize:
		return b.compileGroupBy(parent, v)
	case *dag.Cut:
		b.resetResetters()
		assignments, err := b.compileAssignments(v.Args)
		if err != nil {
			return nil, err
		}
		lhs, rhs := splitAssignments(assignments)
		cutter := expr.NewCutter(b.rctx.Zctx, lhs, rhs)
		if v.Quiet {
			cutter.Quiet()
		}
		return op.NewApplier(b.rctx, parent, cutter, b.resetters), nil
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
		dropper := expr.NewDropper(b.rctx.Zctx, fields)
		return op.NewApplier(b.rctx, parent, dropper, expr.Resetters{}), nil
	case *dag.Sort:
		b.resetResetters()
		fields, err := b.compileExprs(v.Args)
		if err != nil {
			return nil, err
		}
		sort, err := sort.New(b.rctx, parent, fields, v.Order, v.NullsFirst, b.resetters)
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
		return uniq.New(b.rctx, parent, v.Cflag), nil
	case *dag.Pass:
		return pass.New(parent), nil
	case *dag.Filter:
		b.resetResetters()
		f, err := b.compileExpr(v.Expr)
		if err != nil {
			return nil, fmt.Errorf("compiling filter: %w", err)
		}
		return op.NewApplier(b.rctx, parent, expr.NewFilterApplier(b.rctx.Zctx, f), b.resetters), nil
	case *dag.Top:
		b.resetResetters()
		fields, err := b.compileExprs(v.Args)
		if err != nil {
			return nil, fmt.Errorf("compiling top: %w", err)
		}
		return top.New(b.rctx.Zctx, parent, v.Limit, fields, v.Flush, b.resetters), nil
	case *dag.Put:
		b.resetResetters()
		clauses, err := b.compileAssignments(v.Args)
		if err != nil {
			return nil, err
		}
		putter := expr.NewPutter(b.rctx.Zctx, clauses)
		return op.NewApplier(b.rctx, parent, putter, b.resetters), nil
	case *dag.Rename:
		b.resetResetters()
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
		renamer := expr.NewRenamer(b.rctx.Zctx, srcs, dsts)
		return op.NewApplier(b.rctx, parent, renamer, b.resetters), nil
	case *dag.Fuse:
		return fuse.New(b.rctx, parent)
	case *dag.Shape:
		return shape.New(b.rctx, parent)
	case *dag.Join:
		return nil, ErrJoinParents
	case *dag.Merge:
		return nil, errors.New("merge: multiple upstream paths required")
	case *dag.Explode:
		typ, err := zson.ParseType(b.rctx.Zctx, v.Type)
		if err != nil {
			return nil, err
		}
		b.resetResetters()
		args, err := b.compileExprs(v.Args)
		if err != nil {
			return nil, err
		}
		return explode.New(b.rctx.Zctx, parent, args, typ, v.As, b.resetters)
	case *dag.Over:
		return b.compileOver(parent, v)
	case *dag.Yield:
		b.resetResetters()
		exprs, err := b.compileExprs(v.Exprs)
		if err != nil {
			return nil, err
		}
		t := yield.New(parent, exprs, b.resetters)
		return t, nil
	case *dag.PoolScan:
		if parent != nil {
			return nil, errors.New("internal error: pool scan cannot have a parent operator")
		}
		return b.compilePoolScan(v)
	case *dag.PoolMetaScan:
		return meta.NewPoolMetaScanner(b.rctx.Context, b.rctx.Zctx, b.source.Lake(), v.ID, v.Meta)
	case *dag.CommitMetaScan:
		var pruner expr.Evaluator
		if v.Tap && v.KeyPruner != nil {
			var err error
			pruner, err = b.compileExpr(v.KeyPruner)
			if err != nil {
				return nil, err
			}
		}
		return meta.NewCommitMetaScanner(b.rctx.Context, b.rctx.Zctx, b.source.Lake(), v.Pool, v.Commit, v.Meta, pruner)
	case *dag.LakeMetaScan:
		return meta.NewLakeMetaScanner(b.rctx.Context, b.rctx.Zctx, b.source.Lake(), v.Meta)
	case *dag.HTTPScan:
		body := strings.NewReader(v.Body)
		return b.source.OpenHTTP(b.rctx.Context, b.rctx.Zctx, v.URL, v.Format, v.Method, v.Headers, body, demand.All())
	case *dag.FileScan:
		return b.source.Open(b.rctx.Context, b.rctx.Zctx, v.Path, v.Format, b.PushdownOf(v.Filter), demand.All())
	case *dag.DefaultScan:
		pushdown := b.PushdownOf(v.Filter)
		if len(b.readers) == 1 {
			return zbuf.NewScanner(b.rctx.Context, b.readers[0], pushdown)
		}
		scanners := make([]zbuf.Scanner, 0, len(b.readers))
		for _, r := range b.readers {
			scanner, err := zbuf.NewScanner(b.rctx.Context, r, pushdown)
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
			pruner, err = b.compileExpr(v.KeyPruner)
			if err != nil {
				return nil, err
			}
		}
		return meta.NewSortedLister(b.rctx.Context, b.mctx, pool, v.Commit, pruner)
	case *dag.Slicer:
		return meta.NewSlicer(parent, b.mctx), nil
	case *dag.SeqScan:
		pool, err := b.lookupPool(v.Pool)
		if err != nil {
			return nil, err
		}
		var pruner expr.Evaluator
		if v.KeyPruner != nil {
			pruner, err = b.compileExpr(v.KeyPruner)
			if err != nil {
				return nil, err
			}
		}
		return meta.NewSequenceScanner(b.rctx, parent, pool, b.PushdownOf(v.Filter), pruner, b.progress), nil
	case *dag.Deleter:
		pool, err := b.lookupPool(v.Pool)
		if err != nil {
			return nil, err
		}
		var pruner expr.Evaluator
		if v.KeyPruner != nil {
			pruner, err = b.compileExpr(v.KeyPruner)
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
		return meta.NewDeleter(b.rctx, parent, pool, filter, pruner, b.progress, b.deletes), nil
	case *dag.Load:
		return load.New(b.rctx, b.source.Lake(), parent, v.Pool, v.Branch, v.Author, v.Message, v.Meta), nil
	case *dag.Vectorize:
		// If the first op is SeqScan, then pull it out so we can
		// give the scanner a zio.Puller parent (i.e., the lister).
		if scan, ok := v.Body[0].(*dag.SeqScan); ok {
			puller, err := b.compileVamScan(scan, parent)
			if err != nil {
				return nil, err
			}
			if len(v.Body) > 1 {
				p, err := b.compileVamSeq(v.Body[1:], []vector.Puller{puller})
				if err != nil {
					return nil, err
				}
				puller = p[0]
			}
			return vam.NewMaterializer(puller), nil
		}
		//XXX
		return nil, errors.New("dag.Vectorize must begin with SeqScan")
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
	b.resetResetters()
	withNames, withExprs, err := b.compileDefs(over.Defs)
	if err != nil {
		return nil, err
	}
	exprs, err := b.compileExprs(over.Exprs)
	if err != nil {
		return nil, err
	}
	enter := traverse.NewOver(b.rctx, parent, exprs, b.resetters)
	if over.Body == nil {
		return enter, nil
	}
	scope := enter.AddScope(b.rctx.Context, withNames, withExprs)
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
		exit = combine.New(b.rctx, exits)
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
	// Because there can be name collisions between a child and parent scope
	// we clone the current udf map, populate the cloned map, then restore the
	// old scope once the current scope has been built.
	parentUDFs := b.udfs
	b.udfs = maps.Clone(parentUDFs)
	defer func() { b.udfs = parentUDFs }()
	for _, f := range scope.Funcs {
		b.udfs[f.Name] = f.Expr
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
		f = fork.New(b.rctx, parents[0])
	default:
		// Multiple parents: insert a combine followed by a fork for n-way fanout.
		f = fork.New(b.rctx, combine.New(b.rctx, parents))
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

func (b *Builder) compileExprSwitch(swtch *dag.Switch, parents []zbuf.Puller) ([]zbuf.Puller, error) {
	parent := parents[0]
	if len(parents) > 1 {
		parent = combine.New(b.rctx, parents)
	}
	b.resetResetters()
	e, err := b.compileExpr(swtch.Expr)
	if err != nil {
		return nil, err
	}
	s := exprswitch.New(b.rctx, parent, e, b.resetters)
	var exits []zbuf.Puller
	for _, c := range swtch.Cases {
		var val *zed.Value
		if c.Expr != nil {
			val2, err := b.evalAtCompileTime(c.Expr)
			if err != nil {
				return nil, err
			}
			if val2.IsError() {
				return nil, errors.New("switch case is not a constant expression")
			}
			val = &val2
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
		parent = combine.New(b.rctx, parents)
	}
	b.resetResetters()
	var exprs []expr.Evaluator
	for _, c := range swtch.Cases {
		e, err := b.compileExpr(c.Expr)
		if err != nil {
			return nil, fmt.Errorf("compiling switch case filter: %w", err)
		}
		exprs = append(exprs, e)
	}
	switcher := switcher.New(b.rctx, parent, b.resetters)
	var ops []zbuf.Puller
	for i, e := range exprs {
		o, err := b.compileSeq(swtch.Cases[i].Path, []zbuf.Puller{switcher.AddCase(e)})
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
		b.resetResetters()
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
		join, err := join.New(b.rctx, anti, inner, leftParent, rightParent, leftKey, rightKey, leftDir, rightDir, lhs, rhs, b.resetters)
		if err != nil {
			return nil, err
		}
		return []zbuf.Puller{join}, nil
	case *dag.Merge:
		b.resetResetters()
		e, err := b.compileExpr(o.Expr)
		if err != nil {
			return nil, err
		}
		cmp := expr.NewComparator(true, o.Order == order.Desc, e).WithMissingAsNull()
		return []zbuf.Puller{merge.New(b.rctx, parents, cmp.Compare, b.resetters)}, nil
	case *dag.Combine:
		return []zbuf.Puller{combine.New(b.rctx, parents)}, nil
	default:
		var parent zbuf.Puller
		if len(parents) == 1 {
			parent = parents[0]
		} else if len(parents) > 1 {
			parent = combine.New(b.rctx, parents)
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
	l, err := meta.NewSortedLister(b.rctx.Context, b.mctx, pool, scan.Commit, nil)
	if err != nil {
		return nil, err
	}
	slicer := meta.NewSlicer(l, b.mctx)
	return meta.NewSequenceScanner(b.rctx, slicer, pool, nil, nil, b.progress), nil
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
	return b.source.Lake().OpenPool(b.rctx.Context, id)
}

func (b *Builder) evalAtCompileTime(in dag.Expr) (val zed.Value, err error) {
	if in == nil {
		return zed.Null, nil
	}
	e, err := b.compileExpr(in)
	if err != nil {
		return zed.Null, err
	}
	missing := b.zctx().Missing(b.arena)
	// Catch panic as the runtime will panic if there is a
	// reference to a var not in scope, a field access null this, etc.
	defer func() {
		if recover() != nil {
			val = missing
		}
	}()
	return e.Eval(expr.NewContext(b.arena), missing), nil
}

func EvalAtCompileTime(zctx *zed.Context, arena *zed.Arena, in dag.Expr) (zed.Value, error) {
	// We pass in a nil adaptor, which causes a panic for anything adaptor
	// related, which is not currently allowed in an expression sub-query.
	b := NewBuilder(runtime.NewContext(context.Background(), zctx), nil)
	b.arena = arena
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

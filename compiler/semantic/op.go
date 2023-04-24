package semantic

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/data"
	"github.com/brimdata/zed/compiler/kernel"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/pkg/reglob"
	"github.com/brimdata/zed/runtime/expr/function"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
	"golang.org/x/exp/slices"
)

func semFrom(ctx context.Context, scope *Scope, from *ast.From, source *data.Source, head *lakeparse.Commitish) (dag.Op, error) {
	switch len(from.Trunks) {
	case 0:
		return nil, errors.New("internal error: from operator has no paths")
	case 1:
		return semTrunk(ctx, scope, from.Trunks[0], source, head)
	default:
		ops := make([]dag.Op, 0, len(from.Trunks))
		for _, in := range from.Trunks {
			converted, err := semTrunk(ctx, scope, in, source, head)
			if err != nil {
				return nil, err
			}
			ops = append(ops, converted)
		}
		return &dag.Parallel{
			Kind: "Parallel",
			Ops:  ops,
		}, nil
	}
}

func semTrunk(ctx context.Context, scope *Scope, trunk ast.Trunk, ds *data.Source, head *lakeparse.Commitish) (dag.Op, error) {
	if pool, ok := trunk.Source.(*ast.Pool); ok && trunk.Seq != nil {
		switch pool.Spec.Pool.(type) {
		case *ast.Glob, *ast.Regexp:
			return nil, errors.New("=> not allowed after pool pattern in 'from' operator")
		}
	}
	sources, err := semSource(ctx, scope, trunk.Source, ds, head)
	if err != nil {
		return nil, err
	}
	seq, err := semSequential(ctx, scope, trunk.Seq, ds, head)
	if err != nil {
		return nil, err
	}
	if len(sources) == 1 {
		out := &dag.Sequential{Kind: "Sequential", Ops: []dag.Op{sources[0]}}
		if seq != nil {
			out.Ops = append(out.Ops, seq.Ops...)
		}
		return out, nil
	}
	ops := make([]dag.Op, 0, len(sources))
	for _, source := range sources {
		out := &dag.Sequential{Kind: "Sequential", Ops: []dag.Op{source}}
		if seq != nil {
			out.Ops = append(out.Ops, seq.Ops...)
		}
		ops = append(ops, out)
	}
	return &dag.Parallel{Kind: "Parallel", Ops: ops}, nil
}

//XXX make sure you can't read files from a lake instance

func semSource(ctx context.Context, scope *Scope, source ast.Source, ds *data.Source, head *lakeparse.Commitish) ([]dag.Op, error) {
	switch p := source.(type) {
	case *ast.File:
		sortKey, err := semSortKey(p.SortKey)
		if err != nil {
			return nil, err
		}
		return []dag.Op{
			&dag.FileScan{
				Kind:    "FileScan",
				Path:    p.Path,
				Format:  p.Format,
				SortKey: sortKey,
			},
		}, nil
	case *ast.HTTP:
		sortKey, err := semSortKey(p.SortKey)
		if err != nil {
			return nil, err
		}
		return []dag.Op{
			&dag.HTTPScan{
				Kind:    "HTTPScan",
				URL:     p.URL,
				Format:  p.Format,
				SortKey: sortKey,
			},
		}, nil
	case *ast.Pool:
		if !ds.IsLake() {
			return nil, errors.New("semantic analyzer: from pool cannot be used without a lake")
		}
		return semPool(ctx, scope, p, ds, head)
	case *ast.Pass:
		//XXX just connect parent
		return []dag.Op{dag.PassOp}, nil
	case *kernel.Reader:
		// kernel.Reader implements both ast.Source and dag.Op
		return []dag.Op{p}, nil
	default:
		return nil, fmt.Errorf("semantic analyzer: unknown AST source type %T", p)
	}
}

func semSortKey(p *ast.SortKey) (order.SortKey, error) {
	if p == nil || p.Keys == nil {
		return order.Nil, nil
	}
	var keys field.List
	for _, key := range p.Keys {
		this := DotExprToFieldPath(key)
		if this == nil {
			return order.Nil, fmt.Errorf("bad key expr of type %T in file operator", key)
		}
		keys = append(keys, this.Path)
	}
	which, err := order.Parse(p.Order)
	if err != nil {
		return order.Nil, err
	}
	return order.NewSortKey(which, keys), nil
}

func semPool(ctx context.Context, scope *Scope, p *ast.Pool, ds *data.Source, head *lakeparse.Commitish) ([]dag.Op, error) {
	var poolNames []string
	var err error
	switch specPool := p.Spec.Pool.(type) {
	case nil:
		// This is a lake meta-query.
		poolNames = []string{""}
	case *ast.Glob:
		poolNames, err = matchPools(ctx, ds, reglob.Reglob(specPool.Pattern), specPool.Pattern, "glob")
	case *ast.Regexp:
		poolNames, err = matchPools(ctx, ds, specPool.Pattern, specPool.Pattern, "regexp")
	case *ast.String:
		poolNames = []string{specPool.Text}
	default:
		return nil, fmt.Errorf("semantic analyzer: unknown AST pool type %T", specPool)
	}
	if err != nil {
		return nil, err
	}
	var sources []dag.Op
	for _, name := range poolNames {
		source, err := semPoolWithName(ctx, scope, p, name, ds, head)
		if err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	return sources, nil
}

func semPoolWithName(ctx context.Context, scope *Scope, p *ast.Pool, poolName string, ds *data.Source,
	head *lakeparse.Commitish) (dag.Op, error) {
	commit := p.Spec.Commit
	if poolName == "HEAD" {
		if head == nil {
			return nil, errors.New("cannot scan from unknown HEAD")
		}
		poolName = head.Pool
		commit = head.Branch
	}
	if poolName == "" {
		meta := p.Spec.Meta
		if meta == "" {
			return nil, errors.New("pool name missing")
		}
		if _, ok := dag.LakeMetas[meta]; !ok {
			return nil, fmt.Errorf("unknown lake metadata type %q in from operator", meta)
		}
		return &dag.LakeMetaScan{
			Kind: "LakeMetaScan",
			Meta: p.Spec.Meta,
		}, nil
	}
	// If a name appears as an 0x bytes ksuid, convert it to the
	// ksuid string form since the backend doesn't parse the 0x format.
	poolID, err := lakeparse.ParseID(poolName)
	if err != nil {
		poolID, err = ds.PoolID(ctx, poolName)
		if err != nil {
			return nil, err
		}
	}
	if p.At != "" {
		// XXX
		// We no longer use "at" to refer to a commit tag, but if there
		// is no commit tag, we could use an "at" time argument to time
		// travel by going back in the branch log and finding the commit
		// object with the largest time stamp <= the at time.
		// This would require commitRef to be branch name not a commit ID.
		return nil, errors.New("TBD: at clause in from operator needs to use time")
	}
	var commitID ksuid.KSUID
	if commit != "" {
		commitID, err = lakeparse.ParseID(commit)
		if err != nil {
			commitID, err = ds.CommitObject(ctx, poolID, commit)
			if err != nil {
				return nil, err
			}
		}
	}
	if meta := p.Spec.Meta; meta != "" {
		if _, ok := dag.CommitMetas[meta]; ok {
			if commitID == ksuid.Nil {
				commitID, err = ds.CommitObject(ctx, poolID, "main")
				if err != nil {
					return nil, err
				}
			}
			return &dag.CommitMetaScan{
				Kind:   "CommitMetaScan",
				Meta:   meta,
				Pool:   poolID,
				Commit: commitID,
				Tap:    p.Spec.Tap,
			}, nil
		}
		if _, ok := dag.PoolMetas[meta]; ok {
			return &dag.PoolMetaScan{
				Kind: "PoolMetaScan",
				Meta: meta,
				ID:   poolID,
			}, nil
		}
		return nil, fmt.Errorf("unknown metadata type %q in from operator", meta)
	}
	if commitID == ksuid.Nil {
		// This trick here allows us to default to the main branch when
		// there is a "from pool" operator with no meta query or commit object.
		commitID, err = ds.CommitObject(ctx, poolID, "main")
		if err != nil {
			return nil, err
		}
	}
	if p.Delete {
		return &dag.Sequential{
			Kind: "Sequential",
			Ops: []dag.Op{
				&dag.Lister{
					Kind:   "Lister",
					Pool:   poolID,
					Commit: commitID,
				},
				&dag.Deleter{
					Kind: "Deleter",
					Pool: poolID,
				},
			},
		}, nil
	}
	return &dag.PoolScan{
		Kind:   "PoolScan",
		ID:     poolID,
		Commit: commitID,
	}, nil
}

func matchPools(ctx context.Context, ds *data.Source, pattern, origPattern, patternDesc string) ([]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	pools, err := ds.Lake().ListPools(ctx)
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, p := range pools {
		if re.MatchString(p.Name) {
			matches = append(matches, p.Name)
		}
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("%s: pool matching %s not found", origPattern, patternDesc)
	}
	return matches, nil
}

func semSequential(ctx context.Context, scope *Scope, seq *ast.Sequential, ds *data.Source, head *lakeparse.Commitish) (*dag.Sequential, error) {
	if seq == nil {
		return nil, nil
	}
	var ops []dag.Op
	for _, o := range seq.Ops {
		converted, err := semOp(ctx, scope, o, ds, head)
		if err != nil {
			return nil, err
		}
		ops = append(ops, converted)
	}
	return &dag.Sequential{
		Kind: "Sequential",
		Ops:  ops,
	}, nil
}

func semScope(ctx context.Context, scope *Scope, op *ast.Scope, ds *data.Source, head *lakeparse.Commitish) (*dag.Scope, error) {
	scope.Enter()
	defer scope.Exit()
	consts, funcs, err := semDecls(scope, op.Decls)
	if err != nil {
		return nil, err
	}
	seq, err := semSequential(ctx, scope, op.Body, ds, head)
	if err != nil {
		return nil, err
	}
	return &dag.Scope{
		Kind:   "Scope",
		Consts: consts,
		Funcs:  funcs,
		Body:   seq,
	}, nil
}

// semOp does a semantic analysis on a flowgraph to an
// intermediate representation that can be compiled into the runtime
// object.  Currently, it only replaces the group-by duration with
// a bucket call on the ts and replaces FunctionCalls in op context
// with either a group-by or filter op based on the function's name.
func semOp(ctx context.Context, scope *Scope, o ast.Op, ds *data.Source, head *lakeparse.Commitish) (dag.Op, error) {
	switch o := o.(type) {
	case *ast.From:
		return semFrom(ctx, scope, o, ds, head)
	case *ast.Summarize:
		keys, err := semAssignments(scope, o.Keys, true)
		if err != nil {
			return nil, err
		}
		if len(keys) == 0 && len(o.Aggs) == 1 {
			if op := singletonAgg(scope, o.Aggs[0]); op != nil {
				return op, nil
			}
		}
		aggs, err := semAssignments(scope, o.Aggs, true)
		if err != nil {
			return nil, err
		}
		// Note: InputSortDir is copied in here but it's not meaningful
		// coming from a parser AST, only from a worker using the kernel DSL,
		// which is another reason why we need separate parser and kernel ASTs.
		// Said another way, we don't want to do semantic analysis on a worker AST
		// as we presume that work had already been done and we just need
		// to execute it.  For now, the worker only uses a filter expression
		// so this code path isn't hit yet, but it uses this same entry point
		// and it will soon do other stuff so we need to put in place the
		// separation... see issue #2163.
		return &dag.Summarize{
			Kind:  "Summarize",
			Limit: o.Limit,
			Keys:  keys,
			Aggs:  aggs,
		}, nil
	case *ast.Parallel:
		var ops []dag.Op
		for _, o := range o.Ops {
			converted, err := semOp(ctx, scope, o, ds, head)
			if err != nil {
				return nil, err
			}
			ops = append(ops, converted)
		}
		return &dag.Parallel{
			Kind: "Parallel",
			Ops:  ops,
		}, nil
	case *ast.Sequential:
		return semSequential(ctx, scope, o, ds, head)
	case *ast.Scope:
		return semScope(ctx, scope, o, ds, head)
	case *ast.Switch:
		var expr dag.Expr
		if o.Expr != nil {
			var err error
			expr, err = semExpr(scope, o.Expr)
			if err != nil {
				return nil, err
			}
		}
		var cases []dag.Case
		for _, c := range o.Cases {
			var e dag.Expr
			if c.Expr != nil {
				var err error
				e, err = semExpr(scope, c.Expr)
				if err != nil {
					return nil, err
				}
			} else if o.Expr == nil {
				// c.Expr == nil indicates the default case,
				// whose handling depends on p.Expr.
				e = &dag.Literal{
					Kind:  "Literal",
					Value: "true",
				}
			}
			o, err := semOp(ctx, scope, c.Op, ds, head)
			if err != nil {
				return nil, err
			}
			cases = append(cases, dag.Case{Expr: e, Op: o})
		}
		return &dag.Switch{
			Kind:  "Switch",
			Expr:  expr,
			Cases: cases,
		}, nil
	case *ast.Shape:
		return &dag.Shape{Kind: "Shape"}, nil
	case *ast.Cut:
		assignments, err := semAssignments(scope, o.Args, false)
		if err != nil {
			return nil, err
		}
		return &dag.Cut{
			Kind: "Cut",
			Args: assignments,
		}, nil
	case *ast.Drop:
		args, err := semFields(scope, o.Args)
		if err != nil {
			return nil, fmt.Errorf("drop: %w", err)
		}
		if len(args) == 0 {
			return nil, errors.New("drop: no fields given")
		}
		return &dag.Drop{
			Kind: "Drop",
			Args: args,
		}, nil
	case *ast.Sort:
		exprs, err := semExprs(scope, o.Args)
		if err != nil {
			return nil, fmt.Errorf("sort: %w", err)
		}
		return &dag.Sort{
			Kind:       "Sort",
			Args:       exprs,
			Order:      o.Order,
			NullsFirst: o.NullsFirst,
		}, nil
	case *ast.Head:
		expr, err := semExpr(scope, o.Count)
		if err != nil {
			return nil, fmt.Errorf("head: %w", err)
		}
		val, err := kernel.EvalAtCompileTime(scope.zctx, expr)
		if err != nil {
			return nil, fmt.Errorf("head: %w", err)
		}
		if val.AsInt() < 1 {
			return nil, fmt.Errorf("head: expression value is not a positive integer: %s", zson.MustFormatValue(val))
		}
		return &dag.Head{
			Kind:  "Head",
			Count: int(val.AsInt()),
		}, nil
	case *ast.Tail:
		expr, err := semExpr(scope, o.Count)
		if err != nil {
			return nil, fmt.Errorf("tail: %w", err)
		}
		val, err := kernel.EvalAtCompileTime(scope.zctx, expr)
		if err != nil {
			return nil, fmt.Errorf("tail: %w", err)
		}
		if val.AsInt() < 1 {
			return nil, fmt.Errorf("tail: expression value is not a positive integer: %s", zson.MustFormatValue(val))
		}
		return &dag.Tail{
			Kind:  "Tail",
			Count: int(val.AsInt()),
		}, nil
	case *ast.Uniq:
		return &dag.Uniq{
			Kind:  "Uniq",
			Cflag: o.Cflag,
		}, nil
	case *ast.Pass:
		return &dag.Pass{Kind: "Pass"}, nil
	case *ast.OpExpr:
		return semOpExpr(scope, o.Expr)
	case *ast.Search:
		e, err := semExpr(scope, o.Expr)
		if err != nil {
			return nil, err
		}
		return dag.NewFilter(e), nil
	case *ast.Where:
		e, err := semExpr(scope, o.Expr)
		if err != nil {
			return nil, err
		}
		return dag.NewFilter(e), nil
	case *ast.Top:
		args, err := semExprs(scope, o.Args)
		if err != nil {
			return nil, fmt.Errorf("top: %w", err)
		}
		if len(args) == 0 {
			return nil, errors.New("top: no arguments given")
		}
		return &dag.Top{
			Kind:  "Top",
			Args:  args,
			Flush: o.Flush,
			Limit: o.Limit,
		}, nil
	case *ast.Put:
		assignments, err := semAssignments(scope, o.Args, false)
		if err != nil {
			return nil, err
		}
		return &dag.Put{
			Kind: "Put",
			Args: assignments,
		}, nil
	case *ast.OpAssignment:
		return semOpAssignment(scope, o)
	case *ast.Rename:
		var assignments []dag.Assignment
		for _, fa := range o.Args {
			dst, err := semField(scope, fa.LHS)
			if err != nil {
				return nil, errors.New("'rename' requires explicit field references")
			}
			src, err := semField(scope, fa.RHS)
			if err != nil {
				return nil, errors.New("'rename' requires explicit field references")
			}
			if len(dst.Path) != len(src.Path) {
				return nil, fmt.Errorf("cannot rename %s to %s", src, dst)
			}
			// Check that the prefixes match and, if not, report first place
			// that they don't.
			for i := 0; i <= len(src.Path)-2; i++ {
				if src.Path[i] != dst.Path[i] {
					return nil, fmt.Errorf("cannot rename %s to %s (differ in %s vs %s)", src, dst, src.Path[i], dst.Path[i])
				}
			}
			assignments = append(assignments, dag.Assignment{Kind: "Assignment", LHS: dst, RHS: src})
		}
		return &dag.Rename{
			Kind: "Rename",
			Args: assignments,
		}, nil
	case *ast.Fuse:
		return &dag.Fuse{Kind: "Fuse"}, nil
	case *ast.Join:
		rightInput, err := semSequential(ctx, scope, o.RightInput, ds, head)
		if err != nil {
			return nil, err
		}
		leftKey, err := semExpr(scope, o.LeftKey)
		if err != nil {
			return nil, err
		}
		rightKey, err := semExpr(scope, o.RightKey)
		if err != nil {
			return nil, err
		}
		assignments, err := semAssignments(scope, o.Args, false)
		if err != nil {
			return nil, err
		}
		join := &dag.Join{
			Kind:     "Join",
			Style:    o.Style,
			LeftKey:  leftKey,
			RightKey: rightKey,
			Args:     assignments,
		}
		if rightInput != nil {
			par := &dag.Parallel{
				Kind: "Parallel",
				Ops:  []dag.Op{dag.PassOp, rightInput},
			}
			return &dag.Sequential{
				Kind: "Sequential",
				Ops:  []dag.Op{par, join},
			}, nil
		}
		return join, nil
	case *ast.SQLExpr:
		converted, err := convertSQLOp(scope, o)
		if err != nil {
			return nil, err
		}
		// The conversion may be a group-by so we recursively
		// invoke the transformation here...
		if converted == nil {
			return nil, errors.New("unable to covert SQL expression to Zed")
		}
		return converted, nil
	case *ast.Explode:
		typ, err := semType(scope, o.Type)
		if err != nil {
			return nil, err
		}
		args, err := semExprs(scope, o.Args)
		if err != nil {
			return nil, err
		}
		var as dag.Expr
		if o.As == nil {
			as = &dag.This{
				Kind: "This",
				Path: field.Path{"value"},
			}
		} else {
			as, err = semExpr(scope, o.As)
			if err != nil {
				return nil, err
			}
		}
		return &dag.Explode{
			Kind: "Explode",
			Args: args,
			Type: typ,
			As:   as,
		}, nil
	case *ast.Merge:
		expr, err := semExpr(scope, o.Expr)
		if err != nil {
			return nil, fmt.Errorf("merge: %w", err)
		}
		return &dag.Merge{
			Kind:  "Merge",
			Expr:  expr,
			Order: order.Asc, //XXX
		}, nil
	case *ast.Over:
		if len(o.Locals) != 0 && o.Body == nil {
			return nil, errors.New("over operator: cannot have a with clause without a lateral query")
		}
		scope.Enter()
		defer scope.Exit()
		locals, err := semVars(scope, o.Locals)
		if err != nil {
			return nil, err
		}
		exprs, err := semExprs(scope, o.Exprs)
		if err != nil {
			return nil, err
		}
		var seq *dag.Sequential
		if o.Body != nil {
			seq, err = semSequential(ctx, scope, o.Body, ds, head)
			if err != nil {
				return nil, err
			}
		}
		return &dag.Over{
			Kind:  "Over",
			Defs:  locals,
			Exprs: exprs,
			Body:  seq,
		}, nil
	case *ast.Yield:
		exprs, err := semExprs(scope, o.Exprs)
		if err != nil {
			return nil, err
		}
		return &dag.Yield{
			Kind:  "Yield",
			Exprs: exprs,
		}, nil
	case *ast.Load:
		return &dag.Load{
			Kind:    "Load",
			Pool:    o.Pool,
			Branch:  o.Branch,
			Author:  o.Author,
			Message: o.Message,
			Meta:    o.Meta,
		}, nil
	}
	return nil, fmt.Errorf("semantic transform: unknown AST operator type: %T", o)
}

func singletonAgg(scope *Scope, agg ast.Assignment) dag.Op {
	if agg.LHS != nil {
		return nil
	}
	out, err := semAssignment(scope, agg, true)
	if err != nil {
		return nil
	}
	yield := &dag.Yield{
		Kind: "Yield",
	}
	this, ok := out.LHS.(*dag.This)
	if !ok || len(this.Path) != 1 {
		return nil
	}
	yield.Exprs = append(yield.Exprs, this)
	return &dag.Sequential{
		Kind: "Sequential",
		Ops: []dag.Op{
			&dag.Summarize{
				Kind: "Summarize",
				Aggs: []dag.Assignment{out},
			},
			yield,
		},
	}
}

func semDecls(scope *Scope, decls []ast.Decl) ([]dag.Def, []*dag.Func, error) {
	var consts []dag.Def
	var fds []*ast.FuncDecl
	for _, d := range decls {
		switch d := d.(type) {
		case *ast.ConstDecl:
			c, err := semConstDecl(scope, d)
			if err != nil {
				return nil, nil, err
			}
			consts = append(consts, c)
		case *ast.FuncDecl:
			fds = append(fds, d)
		default:
			return nil, nil, fmt.Errorf("invalid declaration type %T", d)
		}
	}
	funcs, err := semFuncDecls(scope, fds)
	if err != nil {
		return nil, nil, err
	}
	return consts, funcs, nil
}

func semConstDecl(scope *Scope, c *ast.ConstDecl) (dag.Def, error) {
	e, err := semExpr(scope, c.Expr)
	if err != nil {
		return dag.Def{}, err
	}
	if err := scope.DefineConst(c.Name, e); err != nil {
		return dag.Def{}, err
	}
	return dag.Def{
		Name: c.Name,
		Expr: e,
	}, nil
}

func semFuncDecls(scope *Scope, decls []*ast.FuncDecl) ([]*dag.Func, error) {
	funcs := make([]*dag.Func, 0, len(decls))
	for _, d := range decls {
		f := &dag.Func{
			Kind:   "Func",
			Name:   d.Name,
			Params: slices.Clone(d.Params),
		}
		if err := scope.DefineFunc(f); err != nil {
			return nil, err
		}
		funcs = append(funcs, f)
	}
	for i, d := range decls {
		var err error
		if funcs[i].Expr, err = semFuncBody(scope, d.Params, d.Expr); err != nil {
			return nil, err
		}
	}
	return funcs, nil
}

func semFuncBody(scope *Scope, params []string, body ast.Expr) (dag.Expr, error) {
	scope.Enter()
	defer scope.Exit()
	for _, p := range params {
		if err := scope.DefineVar(p); err != nil {
			return nil, err
		}
	}
	return semExpr(scope, body)
}

func semVars(scope *Scope, defs []ast.Def) ([]dag.Def, error) {
	var locals []dag.Def
	for _, def := range defs {
		e, err := semExpr(scope, def.Expr)
		if err != nil {
			return nil, err
		}
		name := def.Name
		if err := scope.DefineVar(name); err != nil {
			return nil, err
		}
		locals = append(locals, dag.Def{
			Name: name,
			Expr: e,
		})
	}
	return locals, nil
}

func semOpAssignment(scope *Scope, p *ast.OpAssignment) (dag.Op, error) {
	var aggs, puts []dag.Assignment
	for _, a := range p.Assignments {
		// Parition assignments into agg vs. puts.
		// It's okay to pass false here for the summarize bool because
		// semAssignment will check if the RHS is a dag.Agg and override.
		assignment, err := semAssignment(scope, a, false)
		if err != nil {
			return nil, err
		}
		if _, ok := assignment.RHS.(*dag.Agg); ok {
			aggs = append(aggs, assignment)
		} else {
			puts = append(puts, assignment)
		}
	}
	if len(puts) > 0 && len(aggs) > 0 {
		return nil, errors.New("mix of aggregations and non-aggregations in assignment list")
	}
	if len(puts) > 0 {
		return &dag.Put{
			Kind: "Put",
			Args: puts,
		}, nil
	}
	return &dag.Summarize{
		Kind: "Summarize",
		Aggs: aggs,
	}, nil
}

func semOpExpr(scope *Scope, e ast.Expr) (dag.Op, error) {
	if call, ok := e.(*ast.Call); ok {
		if op, err := semCallOp(scope, call); op != nil || err != nil {
			return op, err
		}
	}
	out, err := semExpr(scope, e)
	if err != nil {
		return nil, err
	}
	if isBool(out) {
		return dag.NewFilter(out), nil
	}
	return &dag.Yield{
		Kind:  "Yield",
		Exprs: []dag.Expr{out},
	}, nil
}

func isBool(e dag.Expr) bool {
	switch e := e.(type) {
	case *dag.Literal:
		return e.Value == "true" || e.Value == "false"
	case *dag.UnaryExpr:
		return isBool(e.Operand)
	case *dag.BinaryExpr:
		switch e.Op {
		case "and", "or", "in", "==", "!=", "<", "<=", ">", ">=":
			return true
		default:
			return false
		}
	case *dag.Conditional:
		return isBool(e.Then) && isBool(e.Else)
	case *dag.Call:
		if e.Name == "cast" {
			if len(e.Args) != 2 {
				return false
			}
			if typval, ok := e.Args[1].(*dag.Literal); ok {
				return typval.Value == "bool"
			}
			return false
		}
		return function.HasBoolResult(e.Name)
	case *dag.Search, *dag.RegexpMatch, *dag.RegexpSearch:
		return true
	default:
		return false
	}
}

func semCallOp(scope *Scope, call *ast.Call) (dag.Op, error) {
	if agg, err := maybeConvertAgg(scope, call); err == nil && agg != nil {
		summarize := &dag.Summarize{
			Kind: "Summarize",
			Aggs: []dag.Assignment{
				{
					Kind: "Assignment",
					LHS:  &dag.This{Kind: "This", Path: field.Path{call.Name}},
					RHS:  agg,
				},
			},
		}
		yield := &dag.Yield{
			Kind:  "Yield",
			Exprs: []dag.Expr{&dag.This{Kind: "This", Path: field.Path{call.Name}}},
		}
		return &dag.Sequential{
			Kind: "Sequential",
			Ops:  []dag.Op{summarize, yield},
		}, nil
	}
	if !function.HasBoolResult(call.Name) {
		return nil, nil
	}
	c, err := semCall(scope, call)
	if err != nil {
		return nil, err
	}
	return dag.NewFilter(c), nil
}

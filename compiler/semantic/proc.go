package semantic

import (
	"context"
	"errors"
	"fmt"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/kernel"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

func semFrom(ctx context.Context, scope *Scope, from *ast.From, adaptor proc.DataAdaptor, head *lakeparse.Commitish) (*dag.From, error) {
	var trunks []dag.Trunk
	for _, in := range from.Trunks {
		converted, err := semTrunk(ctx, scope, in, adaptor, head)
		if err != nil {
			return nil, err
		}
		trunks = append(trunks, converted)
	}
	return &dag.From{
		Kind:   "From",
		Trunks: trunks,
	}, nil
}

func semTrunk(ctx context.Context, scope *Scope, trunk ast.Trunk, adaptor proc.DataAdaptor, head *lakeparse.Commitish) (dag.Trunk, error) {
	source, err := semSource(ctx, scope, trunk.Source, adaptor, head)
	if err != nil {
		return dag.Trunk{}, err
	}
	seq, err := semSequential(ctx, scope, trunk.Seq, adaptor, head)
	if err != nil {
		return dag.Trunk{}, err
	}
	return dag.Trunk{
		Kind:   "Trunk",
		Source: source,
		Seq:    seq,
	}, nil
}

func semSource(ctx context.Context, scope *Scope, source ast.Source, adaptor proc.DataAdaptor, head *lakeparse.Commitish) (dag.Source, error) {
	switch p := source.(type) {
	case *ast.File:
		layout, err := semLayout(p.Layout)
		if err != nil {
			return nil, err
		}
		return &dag.File{
			Kind:   "File",
			Path:   p.Path,
			Format: p.Format,
			Layout: layout,
		}, nil
	case *ast.HTTP:
		layout, err := semLayout(p.Layout)
		if err != nil {
			return nil, err
		}
		return &dag.HTTP{
			Kind:   "HTTP",
			URL:    p.URL,
			Format: p.Format,
			Layout: layout,
		}, nil
	case *ast.Pool:
		return semPool(ctx, scope, p, adaptor, head)
	case *ast.Pass:
		return &dag.Pass{Kind: "Pass"}, nil
	case *kernel.Reader:
		// kernel.Reader implements both ast.Source and dag.Source
		return p, nil
	default:
		return nil, fmt.Errorf("semSource: unknown type %T", p)
	}
}

func semLayout(p *ast.Layout) (order.Layout, error) {
	if p == nil || p.Keys == nil {
		return order.Nil, nil
	}
	var keys field.List
	for _, key := range p.Keys {
		path := DotExprToFieldPath(key)
		if path == nil {
			return order.Nil, fmt.Errorf("bad key expr of type %T in file operator", key)
		}
		keys = append(keys, path.Name)
	}
	which, err := order.Parse(p.Order)
	if err != nil {
		return order.Nil, err
	}
	return order.NewLayout(which, keys), nil
}

func semPool(ctx context.Context, scope *Scope, p *ast.Pool, adaptor proc.DataAdaptor, head *lakeparse.Commitish) (dag.Source, error) {
	poolName := p.Spec.Pool
	commit := p.Spec.Commit
	if poolName == "HEAD" {
		if head == nil {
			return nil, errors.New("cannot scan from unknown HEAD")
		}
		poolName = head.Pool
		commit = head.Branch
	}
	if poolName == "" {
		if p.Spec.Meta == "" {
			return nil, errors.New("pool name missing")
		}
		return &dag.LakeMeta{
			Kind: "LakeMeta",
			Meta: p.Spec.Meta,
		}, nil
	}
	// If a name appears as an 0x bytes ksuid, convert it to the
	// ksuid string form since the backend doesn't parse the 0x format.
	poolID, err := lakeparse.ParseID(poolName)
	if err == nil {
		poolName = poolID.String()
	} else {
		poolID, err = adaptor.PoolID(ctx, poolName)
		if err != nil {
			return nil, err
		}
	}
	var lower, upper dag.Expr
	if r := p.Range; r != nil {
		if r.Lower != nil {
			lower, err = semExpr(scope, r.Lower)
			if err != nil {
				return nil, err
			}
		}
		if r.Upper != nil {
			upper, err = semExpr(scope, r.Upper)
			if err != nil {
				return nil, err
			}
		}
	}
	//var at ksuid.KSUID
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
			commitID, err = adaptor.CommitObject(ctx, poolID, commit)
			if err != nil {
				return nil, err
			}
		}
	}
	if p.Spec.Meta != "" {
		if commitID != ksuid.Nil {
			return &dag.CommitMeta{
				Kind:      "CommitMeta",
				Meta:      p.Spec.Meta,
				Pool:      poolID,
				Commit:    commitID,
				ScanLower: lower,
				ScanUpper: upper,
				ScanOrder: p.ScanOrder,
			}, nil
		}
		return &dag.PoolMeta{
			Kind: "PoolMeta",
			Meta: p.Spec.Meta,
			ID:   poolID,
		}, nil
	}
	if commitID == ksuid.Nil {
		// This trick here allows us to default to the main branch when
		// there is a "from pool" operator with no meta query or commit object.
		commitID, err = adaptor.CommitObject(ctx, poolID, "main")
		if err != nil {
			return nil, err
		}
	}
	return &dag.Pool{
		Kind:      "Pool",
		ID:        poolID,
		Commit:    commitID,
		ScanLower: lower,
		ScanUpper: upper,
		ScanOrder: p.ScanOrder,
	}, nil
}

func semSequential(ctx context.Context, scope *Scope, seq *ast.Sequential, adaptor proc.DataAdaptor, head *lakeparse.Commitish) (*dag.Sequential, error) {
	if seq == nil {
		return nil, nil
	}
	scope.Enter()
	defer scope.Exit()
	consts, err := semConsts(scope, seq.Consts)
	if err != nil {
		return nil, err
	}
	var ops []dag.Op
	for _, p := range seq.Procs {
		converted, err := semProc(ctx, scope, p, adaptor, head)
		if err != nil {
			return nil, err
		}
		ops = append(ops, converted)
	}
	return &dag.Sequential{
		Kind:   "Sequential",
		Consts: consts,
		Ops:    ops,
	}, nil
}

// semProc does a semantic analysis on a flowgraph to an
// intermediate representation that can be compiled into the runtime
// object.  Currently, it only replaces the group-by duration with
// a truncation call on the ts and replaces FunctionCall's in proc context
// with either a group-by or filter-proc based on the function's name.
func semProc(ctx context.Context, scope *Scope, p ast.Proc, adaptor proc.DataAdaptor, head *lakeparse.Commitish) (dag.Op, error) {
	switch p := p.(type) {
	case *ast.From:
		return semFrom(ctx, scope, p, adaptor, head)
	case *ast.Summarize:
		keys, err := semAssignments(scope, p.Keys)
		if err != nil {
			return nil, err
		}
		if duration := p.Duration; duration != nil {
			d, err := nano.ParseDuration(duration.Text)
			if err != nil {
				return nil, err
			}
			durationKey := dag.Assignment{
				Kind: "Assignment",
				LHS: &dag.Path{
					Kind: "Path",
					Name: field.New("ts"),
				},
				RHS: &dag.Call{
					Kind: "Call",
					Name: "trunc",
					Args: []dag.Expr{
						&dag.Path{
							Kind: "Path",
							Name: field.New("ts"),
						},
						&dag.Literal{
							Kind:  "Literal",
							Value: d.String(),
						}},
				},
			}
			keys = append([]dag.Assignment{durationKey}, keys...)
		}
		aggs, err := semAssignments(scope, p.Aggs)
		if err != nil {
			return nil, err
		}
		var dur string
		if p.Duration != nil {
			val, err := zson.ParsePrimitive(p.Duration.Type, p.Duration.Text)
			if err != nil {
				return nil, err
			}
			dur = zson.MustFormatValue(val)
		}
		// Note: InputSortDir is copied in here but it's not meaningful
		// coming from a parser AST, only from a worker using the kernel DSL,
		// which is another reason why we need separate parser and kernel ASTs.
		// Said another way, we don't want to do semantic analysis on a worker AST
		// as we presume that work had already been done and we just need
		// to execute it.  For now, the worker only uses a filter expression
		// so this code path isn't hit yet, but it uses this same entry point
		// and it will soon do other stuff so we need to put in place the
		// separation... see issue #2163.  Also, we copy Duration even though
		// above we changed duration to the a trunc(ts) group-by key as the
		// Duration field is used later by the parallelization operator.
		return &dag.Summarize{
			Kind:     "Summarize",
			Duration: dur,
			Limit:    p.Limit,
			Keys:     keys,
			Aggs:     aggs,
		}, nil
	case *ast.Parallel:
		var ops []dag.Op
		for _, p := range p.Procs {
			converted, err := semProc(ctx, scope, p, adaptor, head)
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
		return semSequential(ctx, scope, p, adaptor, head)
	case *ast.Switch:
		var expr dag.Expr
		if p.Expr != nil {
			var err error
			expr, err = semExpr(scope, p.Expr)
			if err != nil {
				return nil, err
			}
		}
		var cases []dag.Case
		for _, c := range p.Cases {
			var e dag.Expr
			if c.Expr != nil {
				var err error
				e, err = semExpr(scope, c.Expr)
				if err != nil {
					return nil, err
				}
			} else if p.Expr == nil {
				// c.Expr == nil indicates the default case,
				// whose handling depends on p.Expr.
				e = &dag.Literal{
					Kind:  "Literal",
					Value: "true",
				}
			}
			op, err := semProc(ctx, scope, c.Proc, adaptor, head)
			if err != nil {
				return nil, err
			}
			cases = append(cases, dag.Case{Expr: e, Op: op})
		}
		return &dag.Switch{
			Kind:  "Switch",
			Expr:  expr,
			Cases: cases,
		}, nil
	case *ast.Call:
		return convertCallProc(scope, p)
	case *ast.Shape:
		return &dag.Shape{"Shape"}, nil
	case *ast.Cut:
		assignments, err := semAssignments(scope, p.Args)
		if err != nil {
			return nil, err
		}
		return &dag.Cut{
			Kind: "Cut",
			Args: assignments,
		}, nil
	case *ast.Drop:
		args, err := semFields(scope, p.Args)
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
		exprs, err := semExprs(scope, p.Args)
		if err != nil {
			return nil, fmt.Errorf("sort: %w", err)
		}
		return &dag.Sort{
			Kind:       "Sort",
			Args:       exprs,
			Order:      p.Order,
			NullsFirst: p.NullsFirst,
		}, nil
	case *ast.Head:
		limit := p.Count
		if limit == 0 {
			limit = 1
		}
		return &dag.Head{
			Kind:  "Head",
			Count: limit,
		}, nil
	case *ast.Tail:
		limit := p.Count
		if limit == 0 {
			limit = 1
		}
		return &dag.Tail{
			Kind:  "Tail",
			Count: limit,
		}, nil
	case *ast.Uniq:
		return &dag.Uniq{
			Kind:  "Uniq",
			Cflag: p.Cflag,
		}, nil
	case *ast.Pass:
		return &dag.Pass{"Pass"}, nil
	case *ast.Filter:
		e, err := semExpr(scope, p.Expr)
		if err != nil {
			return nil, err
		}
		return &dag.Filter{
			Kind: "Filter",
			Expr: e,
		}, nil
	case *ast.Top:
		args, err := semFields(scope, p.Args)
		if err != nil {
			return nil, fmt.Errorf("top: %w", err)
		}
		if len(args) == 0 {
			return nil, errors.New("top: no arguments given")
		}
		return &dag.Top{
			Kind:  "Top",
			Args:  args,
			Flush: p.Flush,
			Limit: p.Limit,
		}, nil
	case *ast.Put:
		assignments, err := semAssignments(scope, p.Args)
		if err != nil {
			return nil, err
		}
		return &dag.Put{
			Kind: "Put",
			Args: assignments,
		}, nil
	case *ast.OpAssignment:
		return semOpAssignment(scope, p)
	case *ast.Rename:
		var assignments []dag.Assignment
		for _, fa := range p.Args {
			dst, err := semField(scope, fa.LHS)
			if err != nil {
				return nil, err
			}
			src, err := semField(scope, fa.RHS)
			if err != nil {
				return nil, err
			}
			dstField, ok := dst.(*dag.Path)
			if !ok {
				return nil, errors.New("'rename' requires explicit field references")
			}
			srcField, ok := src.(*dag.Path)
			if !ok {
				return nil, errors.New("'rename' requires explicit field references")
			}
			if len(dstField.Name) != len(srcField.Name) {
				return nil, fmt.Errorf("cannot rename %s to %s", src, dst)
			}
			// Check that the prefixes match and, if not, report first place
			// that they don't.
			for i := 0; i <= len(srcField.Name)-2; i++ {
				if srcField.Name[i] != dstField.Name[i] {
					return nil, fmt.Errorf("cannot rename %s to %s (differ in %s vs %s)", srcField, dstField, srcField.Name[i], dstField.Name[i])
				}
			}
			assignments = append(assignments, dag.Assignment{"Assignment", dst, src})
		}
		return &dag.Rename{
			Kind: "Rename",
			Args: assignments,
		}, nil
	case *ast.Fuse:
		return &dag.Fuse{"Fuse"}, nil
	case *ast.Join:
		leftKey, err := semExpr(scope, p.LeftKey)
		if err != nil {
			return nil, err
		}
		rightKey, err := semExpr(scope, p.RightKey)
		if err != nil {
			return nil, err
		}
		assignments, err := semAssignments(scope, p.Args)
		if err != nil {
			return nil, err
		}
		return &dag.Join{
			Kind:     "Join",
			Style:    p.Style,
			LeftKey:  leftKey,
			RightKey: rightKey,
			Args:     assignments,
		}, nil
	case *ast.SQLExpr:
		converted, err := convertSQLProc(scope, p)
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
		typ, err := semType(scope, p.Type)
		if err != nil {
			return nil, err
		}
		args, err := semExprs(scope, p.Args)
		if err != nil {
			return nil, err
		}
		var as dag.Expr
		if p.As == nil {
			as = &dag.Path{
				Kind: "Path",
				Name: field.New("value"),
			}
		} else {
			as, err = semExpr(scope, p.As)
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
		e, err := semField(scope, p.Field)
		if err != nil {
			return nil, err
		}
		field, ok := e.(*dag.Path)
		if !ok || len(field.Name) == 0 {
			//XXX generalize to expr so you can merge this
			return nil, fmt.Errorf("merge: key must be a field")
		}
		return &dag.Merge{
			Kind:  "Merge",
			Key:   field.Name,
			Order: order.Asc, //XXX
		}, nil
	case *ast.Over:
		return semOver(ctx, scope, p, adaptor, head)
	case *ast.Let:
		if p.Over == nil {
			return nil, errors.New("let operator missing traversal in AST")
		}
		if p.Over.Scope == nil {
			return nil, errors.New("let operator missing scope in AST")
		}
		scope.Enter()
		defer scope.Exit()
		locals, err := semVars(scope, p.Locals)
		if err != nil {
			return nil, err
		}
		over, err := semOver(ctx, scope, p.Over, adaptor, head)
		if err != nil {
			return nil, err
		}
		return &dag.Let{
			Kind: "Let",
			Defs: locals,
			Over: over,
		}, nil
	case *ast.Yield:
		exprs, err := semExprs(scope, p.Exprs)
		if err != nil {
			return nil, err
		}
		return &dag.Yield{
			Kind:  "Yield",
			Exprs: exprs,
		}, nil
	}
	return nil, fmt.Errorf("semantic transform: unknown AST type: %v", p)
}

func semOver(ctx context.Context, scope *Scope, in *ast.Over, adaptor proc.DataAdaptor, head *lakeparse.Commitish) (*dag.Over, error) {
	exprs, err := semExprs(scope, in.Exprs)
	if err != nil {
		return nil, err
	}
	var seq *dag.Sequential
	if in.Scope != nil {
		seq, err = semSequential(ctx, scope, in.Scope, adaptor, head)
		if err != nil {
			return nil, err
		}
	}
	return &dag.Over{
		Kind:  "Over",
		Exprs: exprs,
		Scope: seq,
	}, nil
}

func semConsts(scope *Scope, defs []ast.Def) ([]dag.Def, error) {
	var out []dag.Def
	for _, def := range defs {
		e, err := semExpr(scope, def.Expr)
		if err != nil {
			return nil, err
		}
		if err := scope.DefineConst(def.Name, e); err != nil {
			return nil, err
		}
		out = append(out, dag.Def{Name: def.Name, Expr: e})
	}
	return out, nil
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
		// Parition assignments into agg vs. puts
		assignment, err := semAssignment(scope, a)
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

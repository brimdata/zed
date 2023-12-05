package semantic

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/kernel"
	"github.com/brimdata/zed/lakeparse"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/pkg/plural"
	"github.com/brimdata/zed/pkg/reglob"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/runtime/expr/function"
	"github.com/brimdata/zed/zson"
	"github.com/segmentio/ksuid"
)

func (a *analyzer) semSeq(seq ast.Seq) (dag.Seq, error) {
	var converted dag.Seq
	for _, op := range seq {
		var err error
		converted, err = a.semOp(op, converted)
		if err != nil {
			return nil, err
		}
	}
	return converted, nil
}

func (a *analyzer) semFrom(from *ast.From, seq dag.Seq) (dag.Seq, error) {
	switch len(from.Trunks) {
	case 0:
		return nil, errors.New("internal error: from operator has no paths")
	case 1:
		return a.semTrunk(from.Trunks[0], seq)
	default:
		paths := make([]dag.Seq, 0, len(from.Trunks))
		for _, in := range from.Trunks {
			converted, err := a.semTrunk(in, nil)
			if err != nil {
				return nil, err
			}
			paths = append(paths, converted)
		}
		return append(seq, &dag.Fork{
			Kind:  "Fork",
			Paths: paths,
		}), nil
	}
}

func (a *analyzer) semTrunk(trunk ast.Trunk, out dag.Seq) (dag.Seq, error) {
	if pool, ok := trunk.Source.(*ast.Pool); ok && trunk.Seq != nil {
		switch pool.Spec.Pool.(type) {
		case *ast.Glob, *ast.Regexp:
			return nil, errors.New("=> not allowed after pool pattern in 'from' operator")
		}
	}
	sources, err := a.semSource(trunk.Source)
	if err != nil {
		return nil, err
	}
	seq, err := a.semSeq(trunk.Seq)
	if err != nil {
		return nil, err
	}
	if len(sources) == 1 {
		return append(out, append(dag.Seq{sources[0]}, seq...)...), nil
	}
	paths := make([]dag.Seq, 0, len(sources))
	for _, source := range sources {
		paths = append(paths, append(dag.Seq{source}, seq...))
	}
	return append(out, &dag.Fork{Kind: "Fork", Paths: paths}), nil
}

//XXX make sure you can't read files from a lake instance

func (a *analyzer) semSource(source ast.Source) ([]dag.Op, error) {
	switch p := source.(type) {
	case *ast.File:
		sortKey, err := semSortKey(p.SortKey)
		if err != nil {
			return nil, err
		}
		var path string
		switch p := p.Path.(type) {
		case *ast.QuotedString:
			path = p.Text
		case *ast.String:
			// This can be either a reference to a constant or a string.
			if path, err = a.maybeStringConst(p.Text); err != nil {
				return nil, fmt.Errorf("invalid file path: %w", err)
			}
		default:
			return nil, fmt.Errorf("semantic analyzer: unknown AST file type %T", p)
		}
		return []dag.Op{
			&dag.FileScan{
				Kind:    "FileScan",
				Path:    path,
				Format:  p.Format,
				SortKey: sortKey,
			},
		}, nil
	case *ast.HTTP:
		sortKey, err := semSortKey(p.SortKey)
		if err != nil {
			return nil, err
		}
		var headers map[string][]string
		if p.Headers != nil {
			expr, err := a.semExpr(p.Headers)
			if err != nil {
				return nil, err
			}
			val, err := kernel.EvalAtCompileTime(a.zctx, expr)
			if err != nil {
				return nil, fmt.Errorf("headers: %w", err)
			}
			headers, err = unmarshalHeaders(val)
			if err != nil {
				return nil, err
			}
		}
		var url string
		switch p := p.URL.(type) {
		case *ast.QuotedString:
			url = p.Text
		case *ast.String:
			// This can be either a reference to a constant or a string.
			if url, err = a.maybeStringConst(p.Text); err != nil {
				return nil, fmt.Errorf("invalid file path: %w", err)
			}
		default:
			return nil, fmt.Errorf("semantic analyzer: unsupported AST get type %T", p)
		}
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			return nil, fmt.Errorf("get: invalid URL %s", url)
		}
		return []dag.Op{
			&dag.HTTPScan{
				Kind:    "HTTPScan",
				URL:     url,
				Format:  p.Format,
				SortKey: sortKey,
				Method:  p.Method,
				Headers: headers,
				Body:    p.Body,
			},
		}, nil
	case *ast.Pool:
		if !a.source.IsLake() {
			return nil, errors.New("semantic analyzer: from pool cannot be used without a lake")
		}
		return a.semPool(p)
	case *ast.Pass:
		//XXX just connect parent
		return []dag.Op{dag.PassOp}, nil
	default:
		return nil, fmt.Errorf("semantic analyzer: unknown AST source type %T", p)
	}
}

func unmarshalHeaders(val *zed.Value) (map[string][]string, error) {
	if !zed.IsRecordType(val.Type) {
		return nil, errors.New("headers value must be a record")
	}
	headers := map[string][]string{}
	for i, f := range val.Fields() {
		if inner := zed.InnerType(f.Type); inner == nil || inner.ID() != zed.IDString {
			return nil, errors.New("headers field value must be an array or set of strings")
		}
		fieldVal := val.DerefByColumn(i)
		if fieldVal == nil {
			continue
		}
		for it := fieldVal.Iter(); !it.Done(); {
			if b := it.Next(); b != nil {
				headers[f.Name] = append(headers[f.Name], zed.DecodeString(b))
			}
		}
	}
	return headers, nil
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

func (a *analyzer) semPool(p *ast.Pool) ([]dag.Op, error) {
	var poolNames []string
	var err error
	switch specPool := p.Spec.Pool.(type) {
	case nil:
		// This is a lake meta-query.
		poolNames = []string{""}
	case *ast.Glob:
		poolNames, err = a.matchPools(reglob.Reglob(specPool.Pattern), specPool.Pattern, "glob")
	case *ast.Regexp:
		poolNames, err = a.matchPools(specPool.Pattern, specPool.Pattern, "regexp")
	case *ast.String:
		// This can be either a reference to a constant or a string.
		name, err := a.maybeStringConst(specPool.Text)
		if err != nil {
			return nil, fmt.Errorf("invalid pool name: %w", err)
		}
		poolNames = []string{name}
	case *ast.QuotedString:
		poolNames = []string{specPool.Text}
	default:
		return nil, fmt.Errorf("semantic analyzer: unknown AST pool type %T", specPool)
	}
	if err != nil {
		return nil, err
	}
	var sources []dag.Op
	for _, name := range poolNames {
		source, err := a.semPoolWithName(p, name)
		if err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	return sources, nil
}

func (a *analyzer) maybeStringConst(name string) (string, error) {
	e, err := a.scope.LookupExpr(name)
	if err != nil || e == nil {
		return name, err
	}
	l, ok := e.(*dag.Literal)
	if !ok {
		return "", fmt.Errorf("%s: string value required", name)
	}
	val := zson.MustParseValue(a.zctx, l.Value)
	if val.Type.ID() != zed.IDString {
		return "", fmt.Errorf("%s: string value required", name)
	}
	return val.AsString(), nil
}

func (a *analyzer) semPoolWithName(p *ast.Pool, poolName string) (dag.Op, error) {
	commit := p.Spec.Commit
	if poolName == "HEAD" {
		if a.head == nil {
			return nil, errors.New("cannot scan from unknown HEAD")
		}
		poolName = a.head.Pool
		commit = a.head.Branch
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
		poolID, err = a.source.PoolID(a.ctx, poolName)
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
			commitID, err = a.source.CommitObject(a.ctx, poolID, commit)
			if err != nil {
				return nil, err
			}
		}
	}
	if meta := p.Spec.Meta; meta != "" {
		if _, ok := dag.CommitMetas[meta]; ok {
			if commitID == ksuid.Nil {
				commitID, err = a.source.CommitObject(a.ctx, poolID, "main")
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
		commitID, err = a.source.CommitObject(a.ctx, poolID, "main")
		if err != nil {
			return nil, err
		}
	}
	if p.Delete {
		return &dag.DeleteScan{
			Kind:   "DeleteScan",
			ID:     poolID,
			Commit: commitID,
		}, nil
	}
	return &dag.PoolScan{
		Kind:   "PoolScan",
		ID:     poolID,
		Commit: commitID,
	}, nil
}

func (a *analyzer) matchPools(pattern, origPattern, patternDesc string) ([]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	pools, err := a.source.Lake().ListPools(a.ctx)
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

func (a *analyzer) semScope(op *ast.Scope) (*dag.Scope, error) {
	a.scope = NewScope(a.scope)
	defer a.exitScope()
	consts, funcs, err := a.semDecls(op.Decls)
	if err != nil {
		return nil, err
	}
	body, err := a.semSeq(op.Body)
	if err != nil {
		return nil, err
	}
	return &dag.Scope{
		Kind:   "Scope",
		Consts: consts,
		Funcs:  funcs,
		Body:   body,
	}, nil
}

// semOp does a semantic analysis on a flowgraph to an
// intermediate representation that can be compiled into the runtime
// object.  Currently, it only replaces the group-by duration with
// a bucket call on the ts and replaces FunctionCalls in op context
// with either a group-by or filter op based on the function's name.
func (a *analyzer) semOp(o ast.Op, seq dag.Seq) (dag.Seq, error) {
	switch o := o.(type) {
	case *ast.From:
		return a.semFrom(o, seq)
	case *ast.Summarize:
		keys, err := a.semAssignments(o.Keys)
		if err != nil {
			return nil, err
		}
		if assignmentHasDynamicLHS(keys) {
			return nil, errors.New("summarize: key output field must be static")
		}
		if len(keys) == 0 && len(o.Aggs) == 1 {
			if seq := a.singletonAgg(o.Aggs[0], seq); seq != nil {
				return seq, nil
			}
		}
		aggs, err := a.semAssignments(o.Aggs)
		if err != nil {
			return nil, err
		}
		if assignmentHasDynamicLHS(aggs) {
			return nil, errors.New("summarize: aggregate output field must be static")
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
		return append(seq, &dag.Summarize{
			Kind:  "Summarize",
			Limit: o.Limit,
			Keys:  keys,
			Aggs:  aggs,
		}), nil
	case *ast.Parallel:
		var paths []dag.Seq
		for _, seq := range o.Paths {
			converted, err := a.semSeq(seq)
			if err != nil {
				return nil, err
			}
			paths = append(paths, converted)
		}
		return append(seq, &dag.Fork{
			Kind:  "Fork",
			Paths: paths,
		}), nil
	case *ast.Scope:
		op, err := a.semScope(o)
		if err != nil {
			return nil, err
		}
		return append(seq, op), nil
	case *ast.Switch:
		var expr dag.Expr
		if o.Expr != nil {
			var err error
			expr, err = a.semExpr(o.Expr)
			if err != nil {
				return nil, err
			}
		}
		var cases []dag.Case
		for _, c := range o.Cases {
			var e dag.Expr
			if c.Expr != nil {
				var err error
				e, err = a.semExpr(c.Expr)
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
			path, err := a.semSeq(c.Path)
			if err != nil {
				return nil, err
			}
			cases = append(cases, dag.Case{Expr: e, Path: path})
		}
		return append(seq, &dag.Switch{
			Kind:  "Switch",
			Expr:  expr,
			Cases: cases,
		}), nil
	case *ast.Shape:
		return append(seq, &dag.Shape{Kind: "Shape"}), nil
	case *ast.Cut:
		assignments, err := a.semAssignments(o.Args)
		if err != nil {
			return nil, err
		}
		// Collect static paths so we can check on what is available.
		var fields field.List
		for _, a := range assignments {
			if this, ok := a.LHS.(*dag.This); ok {
				fields = append(fields, this.Path)
			}
		}
		if _, err = zed.NewRecordBuilder(a.zctx, fields); err != nil {
			return nil, fmt.Errorf("cut: %w", err)
		}
		return append(seq, &dag.Cut{
			Kind: "Cut",
			Args: assignments,
		}), nil
	case *ast.Drop:
		args, err := a.semFields(o.Args)
		if err != nil {
			return nil, fmt.Errorf("drop: %w", err)
		}
		if len(args) == 0 {
			return nil, errors.New("drop: no fields given")
		}
		return append(seq, &dag.Drop{
			Kind: "Drop",
			Args: args,
		}), nil
	case *ast.Sort:
		exprs, err := a.semExprs(o.Args)
		if err != nil {
			return nil, fmt.Errorf("sort: %w", err)
		}
		return append(seq, &dag.Sort{
			Kind:       "Sort",
			Args:       exprs,
			Order:      o.Order,
			NullsFirst: o.NullsFirst,
		}), nil
	case *ast.Head:
		expr, err := a.semExpr(o.Count)
		if err != nil {
			return nil, fmt.Errorf("head: %w", err)
		}
		val, err := kernel.EvalAtCompileTime(a.zctx, expr)
		if err != nil {
			return nil, fmt.Errorf("head: %w", err)
		}
		if val.AsInt() < 1 {
			return nil, fmt.Errorf("head: expression value is not a positive integer: %s", zson.FormatValue(val))
		}
		return append(seq, &dag.Head{
			Kind:  "Head",
			Count: int(val.AsInt()),
		}), nil
	case *ast.Tail:
		expr, err := a.semExpr(o.Count)
		if err != nil {
			return nil, fmt.Errorf("tail: %w", err)
		}
		val, err := kernel.EvalAtCompileTime(a.zctx, expr)
		if err != nil {
			return nil, fmt.Errorf("tail: %w", err)
		}
		if val.AsInt() < 1 {
			return nil, fmt.Errorf("tail: expression value is not a positive integer: %s", zson.FormatValue(val))
		}
		return append(seq, &dag.Tail{
			Kind:  "Tail",
			Count: int(val.AsInt()),
		}), nil
	case *ast.Uniq:
		return append(seq, &dag.Uniq{
			Kind:  "Uniq",
			Cflag: o.Cflag,
		}), nil
	case *ast.Pass:
		return append(seq, dag.PassOp), nil
	case *ast.OpExpr:
		return a.semOpExpr(o.Expr, seq)
	case *ast.Search:
		e, err := a.semExpr(o.Expr)
		if err != nil {
			return nil, err
		}
		return append(seq, dag.NewFilter(e)), nil
	case *ast.Where:
		e, err := a.semExpr(o.Expr)
		if err != nil {
			return nil, err
		}
		return append(seq, dag.NewFilter(e)), nil
	case *ast.Top:
		args, err := a.semExprs(o.Args)
		if err != nil {
			return nil, fmt.Errorf("top: %w", err)
		}
		if len(args) == 0 {
			return nil, errors.New("top: no arguments given")
		}
		return append(seq, &dag.Top{
			Kind:  "Top",
			Args:  args,
			Flush: o.Flush,
			Limit: o.Limit,
		}), nil
	case *ast.Put:
		assignments, err := a.semAssignments(o.Args)
		if err != nil {
			return nil, err
		}
		// We can do collision checking on static paths, so check what we can.
		var fields field.List
		for _, a := range assignments {
			if this, ok := a.LHS.(*dag.This); ok {
				fields = append(fields, this.Path)
			}
		}
		if err := expr.CheckPutFields(fields); err != nil {
			return nil, fmt.Errorf("put: %w", err)
		}
		return append(seq, &dag.Put{
			Kind: "Put",
			Args: assignments,
		}), nil
	case *ast.OpAssignment:
		converted, err := a.semOpAssignment(o)
		if err != nil {
			return nil, err
		}
		return append(seq, converted), nil
	case *ast.Rename:
		var assignments []dag.Assignment
		for _, fa := range o.Args {
			assign, err := a.semAssignment(fa)
			if err != nil {
				return nil, fmt.Errorf("rename: %w", err)
			}
			if !isLval(assign.RHS) {
				return nil, fmt.Errorf("rename: illegal right-hand side of assignment")
			}
			// If both paths are static validate them. Otherwise this will be
			// done at runtime.
			lhs, lhsOk := assign.LHS.(*dag.This)
			rhs, rhsOk := assign.RHS.(*dag.This)
			if rhsOk && lhsOk {
				if err := expr.CheckRenameField(lhs.Path, rhs.Path); err != nil {
					return nil, fmt.Errorf("rename: %w", err)
				}
			}
			assignments = append(assignments, assign)
		}
		return append(seq, &dag.Rename{
			Kind: "Rename",
			Args: assignments,
		}), nil
	case *ast.Fuse:
		return append(seq, &dag.Fuse{Kind: "Fuse"}), nil
	case *ast.Join:
		rightInput, err := a.semSeq(o.RightInput)
		if err != nil {
			return nil, err
		}
		leftKey, err := a.semExpr(o.LeftKey)
		if err != nil {
			return nil, err
		}
		rightKey, err := a.semExpr(o.RightKey)
		if err != nil {
			return nil, err
		}
		assignments, err := a.semAssignments(o.Args)
		if err != nil {
			return nil, err
		}
		join := &dag.Join{
			Kind:     "Join",
			Style:    o.Style,
			LeftDir:  order.Unknown,
			LeftKey:  leftKey,
			RightDir: order.Unknown,
			RightKey: rightKey,
			Args:     assignments,
		}
		if rightInput != nil {
			par := &dag.Fork{
				Kind:  "Fork",
				Paths: []dag.Seq{{dag.PassOp}, rightInput},
			}
			seq = append(seq, par)
		}
		return append(seq, join), nil
	case *ast.SQLExpr:
		var err error
		seq, err = a.convertSQLOp(o, seq)
		if err != nil {
			return nil, err
		}
		// The conversion may be a group-by so we recursively
		// invoke the transformation here...
		if seq == nil {
			return nil, errors.New("unable to covert SQL expression to Zed")
		}
		return seq, nil
	case *ast.Explode:
		typ, err := a.semType(o.Type)
		if err != nil {
			return nil, err
		}
		args, err := a.semExprs(o.Args)
		if err != nil {
			return nil, err
		}
		var as string
		if o.As == nil {
			as = "value"
		} else {
			e, err := a.semExpr(o.As)
			if err != nil {
				return nil, err
			}
			this, ok := e.(*dag.This)
			if !ok {
				return nil, errors.New("explode: as clause must be a field reference")
			} else if len(this.Path) != 1 {
				return nil, errors.New("explode: field must be a top-level field")
			}
			as = this.Path[0]
		}
		return append(seq, &dag.Explode{
			Kind: "Explode",
			Args: args,
			Type: typ,
			As:   as,
		}), nil
	case *ast.Merge:
		expr, err := a.semExpr(o.Expr)
		if err != nil {
			return nil, fmt.Errorf("merge: %w", err)
		}
		return append(seq, &dag.Merge{
			Kind:  "Merge",
			Expr:  expr,
			Order: order.Asc, //XXX
		}), nil
	case *ast.Over:
		if len(o.Locals) != 0 && o.Body == nil {
			return nil, errors.New("over operator: cannot have a with clause without a lateral query")
		}
		a.enterScope()
		defer a.exitScope()
		locals, err := a.semVars(o.Locals)
		if err != nil {
			return nil, err
		}
		exprs, err := a.semExprs(o.Exprs)
		if err != nil {
			return nil, err
		}
		var body dag.Seq
		if o.Body != nil {
			body, err = a.semSeq(o.Body)
			if err != nil {
				return nil, err
			}
		}
		return append(seq, &dag.Over{
			Kind:  "Over",
			Defs:  locals,
			Exprs: exprs,
			Body:  body,
		}), nil
	case *ast.Sample:
		e, err := a.semExpr(o.Expr)
		if err != nil {
			return nil, err
		}
		seq = append(seq, &dag.Summarize{
			Kind: "Summarize",
			Aggs: []dag.Assignment{
				{
					Kind: "Assignment",
					LHS:  pathOf("sample"),
					RHS:  &dag.Agg{Kind: "Agg", Name: "any", Expr: e},
				},
			},
			Keys: []dag.Assignment{
				{
					Kind: "Assignment",
					LHS:  pathOf("shape"),
					RHS:  &dag.Call{Kind: "Call", Name: "typeof", Args: []dag.Expr{e}},
				},
			},
		})
		return append(seq, &dag.Yield{
			Kind:  "Yield",
			Exprs: []dag.Expr{&dag.This{Kind: "This", Path: field.Path{"sample"}}},
		}), nil
	case *ast.Yield:
		exprs, err := a.semExprs(o.Exprs)
		if err != nil {
			return nil, err
		}
		return append(seq, &dag.Yield{
			Kind:  "Yield",
			Exprs: exprs,
		}), nil
	case *ast.Load:
		poolID, err := lakeparse.ParseID(o.Pool)
		if err != nil {
			poolID, err = a.source.PoolID(a.ctx, o.Pool)
			if err != nil {
				return nil, err
			}
		}
		return append(seq, &dag.Load{
			Kind:    "Load",
			Pool:    poolID,
			Branch:  o.Branch,
			Author:  o.Author,
			Message: o.Message,
			Meta:    o.Meta,
		}), nil
	}
	return nil, fmt.Errorf("semantic transform: unknown AST operator type: %T", o)
}

func (a *analyzer) singletonAgg(agg ast.Assignment, seq dag.Seq) dag.Seq {
	if agg.LHS != nil {
		return nil
	}
	out, err := a.semAssignment(agg)
	if err != nil {
		return nil
	}
	this, ok := out.LHS.(*dag.This)
	if !ok || len(this.Path) != 1 {
		return nil
	}
	return append(seq,
		&dag.Summarize{
			Kind: "Summarize",
			Aggs: []dag.Assignment{out},
		},
		&dag.Yield{
			Kind:  "Yield",
			Exprs: []dag.Expr{this},
		},
	)
}

func (a *analyzer) semDecls(decls []ast.Decl) ([]dag.Def, []*dag.Func, error) {
	var consts []dag.Def
	var fnDecls []*ast.FuncDecl
	for _, d := range decls {
		switch d := d.(type) {
		case *ast.ConstDecl:
			c, err := a.semConstDecl(d)
			if err != nil {
				return nil, nil, err
			}
			consts = append(consts, c)
		case *ast.FuncDecl:
			fnDecls = append(fnDecls, d)
		case *ast.OpDecl:
			if err := a.semOpDecl(d); err != nil {
				return nil, nil, err
			}
		default:
			return nil, nil, fmt.Errorf("invalid declaration type %T", d)
		}
	}
	funcs, err := a.semFuncDecls(fnDecls)
	if err != nil {
		return nil, nil, err
	}
	return consts, funcs, nil
}

func (a *analyzer) semConstDecl(c *ast.ConstDecl) (dag.Def, error) {
	e, err := a.semExpr(c.Expr)
	if err != nil {
		return dag.Def{}, err
	}
	if err := a.scope.DefineConst(a.zctx, c.Name, e); err != nil {
		return dag.Def{}, err
	}
	return dag.Def{
		Name: c.Name,
		Expr: e,
	}, nil
}

func (a *analyzer) semFuncDecls(decls []*ast.FuncDecl) ([]*dag.Func, error) {
	funcs := make([]*dag.Func, 0, len(decls))
	for _, d := range decls {
		f := &dag.Func{
			Kind:   "Func",
			Name:   d.Name,
			Params: slices.Clone(d.Params),
		}
		if err := a.scope.DefineAs(f.Name, f); err != nil {
			return nil, err
		}
		funcs = append(funcs, f)
	}
	for i, d := range decls {
		var err error
		if funcs[i].Expr, err = a.semFuncBody(d.Params, d.Expr); err != nil {
			return nil, err
		}
	}
	return funcs, nil
}

func (a *analyzer) semFuncBody(params []string, body ast.Expr) (dag.Expr, error) {
	a.enterScope()
	defer a.exitScope()
	for _, p := range params {
		if err := a.scope.DefineVar(p); err != nil {
			return nil, err
		}
	}
	return a.semExpr(body)
}

func (a *analyzer) semOpDecl(d *ast.OpDecl) error {
	m := make(map[string]bool)
	for _, p := range d.Params {
		if m[p] {
			return fmt.Errorf("%s: duplicate parameter %q", d.Name, p)
		}
		m[p] = true
	}
	return a.scope.DefineAs(d.Name, &opDecl{ast: d, scope: a.scope})
}

func (a *analyzer) semVars(defs []ast.Def) ([]dag.Def, error) {
	var locals []dag.Def
	for _, def := range defs {
		e, err := a.semExpr(def.Expr)
		if err != nil {
			return nil, err
		}
		name := def.Name
		if err := a.scope.DefineVar(name); err != nil {
			return nil, err
		}
		locals = append(locals, dag.Def{
			Name: name,
			Expr: e,
		})
	}
	return locals, nil
}

func (a *analyzer) semOpAssignment(p *ast.OpAssignment) (dag.Op, error) {
	var aggs, puts []dag.Assignment
	for _, assign := range p.Assignments {
		// Parition assignments into agg vs. puts.
		a, err := a.semAssignment(assign)
		if err != nil {
			return nil, err
		}
		if _, ok := a.RHS.(*dag.Agg); ok {
			if _, ok := a.LHS.(*dag.This); !ok {
				return nil, errors.New("summarize: aggregate output field must be static")
			}
			aggs = append(aggs, a)
		} else {
			puts = append(puts, a)
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

func assignmentHasDynamicLHS(assignments []dag.Assignment) bool {
	for _, a := range assignments {
		if _, ok := a.LHS.(*dag.This); !ok {
			return true
		}
	}
	return false
}

func (a *analyzer) semOpExpr(e ast.Expr, seq dag.Seq) (dag.Seq, error) {
	if call, ok := e.(*ast.Call); ok {
		if seq, err := a.semCallOp(call, seq); seq != nil || err != nil {
			return seq, err
		}
	}
	out, err := a.semExpr(e)
	if err != nil {
		return nil, err
	}
	if a.isBool(out) {
		return append(seq, dag.NewFilter(out)), nil
	}
	return append(seq, &dag.Yield{
		Kind:  "Yield",
		Exprs: []dag.Expr{out},
	}), nil
}

func (a *analyzer) isBool(e dag.Expr) bool {
	switch e := e.(type) {
	case *dag.Literal:
		return e.Value == "true" || e.Value == "false"
	case *dag.UnaryExpr:
		return a.isBool(e.Operand)
	case *dag.BinaryExpr:
		switch e.Op {
		case "and", "or", "in", "==", "!=", "<", "<=", ">", ">=":
			return true
		default:
			return false
		}
	case *dag.Conditional:
		return a.isBool(e.Then) && a.isBool(e.Else)
	case *dag.Call:
		// If udf recurse to inner expression.
		if f, _ := a.scope.LookupExpr(e.Name); f != nil {
			return a.isBool(f.(*dag.Func).Expr)
		}
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

func (a *analyzer) semCallOp(call *ast.Call, seq dag.Seq) (dag.Seq, error) {
	if body, err := a.maybeConvertUserOp(call, seq); err != nil {
		return nil, err
	} else if body != nil {
		return append(seq, body...), nil
	}
	if agg, err := a.maybeConvertAgg(call); err == nil && agg != nil {
		summarize := &dag.Summarize{
			Kind: "Summarize",
			Aggs: []dag.Assignment{
				{
					Kind: "Assignment",
					LHS:  pathOf(call.Name),
					RHS:  agg,
				},
			},
		}
		yield := &dag.Yield{
			Kind:  "Yield",
			Exprs: []dag.Expr{&dag.This{Kind: "This", Path: field.Path{call.Name}}},
		}
		return append(append(seq, summarize), yield), nil
	}
	if !function.HasBoolResult(call.Name) {
		return nil, nil
	}
	c, err := a.semCall(call)
	if err != nil {
		return nil, err
	}
	return append(seq, dag.NewFilter(c)), nil
}

// maybeConvertUserOp returns nil, nil if the call is determined to not be a
// UserOp, otherwise it returns the compiled op or the encountered error.
func (a *analyzer) maybeConvertUserOp(call *ast.Call, seq dag.Seq) (dag.Seq, error) {
	decl, err := a.scope.lookupOp(call.Name)
	if decl == nil || err != nil {
		return nil, nil
	}
	if call.Where != nil {
		return nil, fmt.Errorf("%s(): user defined operators cannot have a where clause", call.Name)
	}
	params, args := decl.ast.Params, call.Args
	if len(params) != len(args) {
		return nil, fmt.Errorf("%s(): %d arg%s provided when operator expects %d arg%s", call.Name, len(params), plural.Slice(params, "s"), len(args), plural.Slice(args, "s"))
	}
	exprs := make([]dag.Expr, len(decl.ast.Params))
	for i, arg := range args {
		e, err := a.semExpr(arg)
		if err != nil {
			return nil, err
		}
		// Transform non-path arguments into literals.
		if _, ok := e.(*dag.This); !ok {
			val, err := kernel.EvalAtCompileTime(a.zctx, e)
			if err != nil {
				return nil, err
			}
			if val.IsError() {
				if val.IsMissing() {
					return nil, fmt.Errorf("%q: non-path arguments cannot have variable dependency", decl.ast.Params[i])
				} else {
					return nil, fmt.Errorf("%q: %q", decl.ast.Params[i], string(val.Bytes()))
				}
			}
			e = &dag.Literal{
				Kind:  "Literal",
				Value: zson.FormatValue(val),
			}
		}
		exprs[i] = e
	}
	if slices.Contains(a.opStack, decl.ast) {
		return nil, opCycleError(append(a.opStack, decl.ast))
	}
	a.opStack = append(a.opStack, decl.ast)
	oldscope := a.scope
	a.scope = NewScope(decl.scope)
	defer func() {
		a.opStack = a.opStack[:len(a.opStack)-1]
		a.scope = oldscope
	}()
	for i, p := range params {
		if err := a.scope.DefineAs(p, exprs[i]); err != nil {
			return nil, err
		}
	}
	return a.semSeq(decl.ast.Body)
}

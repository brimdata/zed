package semantic

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/compiler/ast"
	"github.com/brimsec/zq/compiler/kernel"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/proc/combine"
	"github.com/brimsec/zq/proc/join"
	"github.com/brimsec/zq/proc/merge"
	"github.com/brimsec/zq/proc/split"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

func Analyze(program *ast.Program) (*kernel.Program, error) {
	//XXX TBD
	return nil, nil
}

var ErrJoinParents = errors.New("join requires two upstream parallel query paths")

type Hook func(ast.Proc, *proc.Context, proc.Interface) (proc.Interface, error)

func isContainerProc(node ast.Proc) bool {
	if _, ok := node.(*ast.SequentialProc); ok {
		return true
	}
	if _, ok := node.(*ast.ParallelProc); ok {
		return true
	}
	return false
}

func semAssignments(zctx *resolver.Context, scope *kernel.Scope, assignemnts []ast.Assignment) ([]kernel.Assignment, error) {
	return nil, nil
}

func semAggFuncs(assignemnts []ast.Assignment) ([]kernel.AggAssignment, error) {
	return nil, nil

}

func semExprs(zctx *resolver.Context, scope *kernel.Scope, exprs []ast.Expression) ([]kernel.Expr, error) {
	return nil, nil
}

//XXX should return kernel.Boolean interface
func semBool(zctx *resolver.Context, scope *kernel.Scope, predicate ast.Expression) (kernel.Expr, error) {
	return nil, nil
}

func semField(expr ast.Expression) (field.Static, error) {
	//XXX see CompilLval
	return nil, nil
}

func semProc(p ast.Proc, pctx *proc.Context, scope *kernel.Scope) (kernel.Operator, error) {
	switch p := p.(type) {
	case *ast.GroupByProc:
		keys, err := semAssignments(pctx.TypeContext, scope, p.Keys)
		if err != nil {
			return nil, err
		}
		aggs, err := semAggFuncs(p.Reducers)
		if err != nil {
			return nil, err
		}
		duration := nano.Duration(int64(p.Duration.Seconds), 0)
		bytes := zng.EncodeDuration(duration)
		return &kernel.Agg{
			Op:       "Agg",
			Keys:     keys,
			Aggs:     aggs,
			Duration: zng.Value{zng.TypeDuration, bytes},
			Limit:    p.Limit,
		}, nil

	case *ast.CutProc:
		a, err := semAssignments(pctx.TypeContext, scope, p.Fields)
		if err != nil {
			return nil, err
		}
		return &kernel.Cut{
			Op:          "Cut",
			Assignments: a,
		}, nil

	case *ast.PickProc:
		a, err := semAssignments(pctx.TypeContext, scope, p.Fields)
		if err != nil {
			return nil, err
		}
		return &kernel.Pick{
			Op:          "Pick",
			Assignments: a,
		}, nil

	case *ast.DropProc:
		fields := make([]field.Static, 0, len(p.Fields))
		for _, e := range p.Fields {
			field, ok := ast.DotExprToField(e)
			if !ok {
				return nil, errors.New("drop: arg not a field")
			}
			fields = append(fields, field)
		}
		return &kernel.Drop{
			Op:     "Drop",
			Fields: fields,
		}, nil

	case *ast.SortProc:
		fields, err := semExprs(pctx.TypeContext, scope, p.Fields)
		if err != nil {
			return nil, err
		}
		return &kernel.Sort{
			Op:     "Sort",
			Fields: fields,
		}, nil

	case *ast.HeadProc:
		count := p.Count
		if count == 0 {
			count = 1
		}
		return &kernel.Head{
			Op:    "Head",
			Count: count,
		}, nil

	case *ast.TailProc:
		count := p.Count
		if count == 0 {
			count = 1
		}
		return &kernel.Tail{
			Op:    "Tail",
			Count: count,
		}, nil

	case *ast.UniqProc:
		return &kernel.Uniq{
			Op:    "Uniq",
			Cflag: p.Cflag,
		}, nil

		//XXX delete this
	case *ast.PassProc:
		return &kernel.Pass{"Pass"}, nil

	case *ast.FilterProc:
		f, err := semBool(pctx.TypeContext, scope, p.Filter)
		if err != nil {
			return nil, err
		}
		return &kernel.Filter{
			Op:        "Filter",
			Predicate: f,
		}, nil

	case *ast.TopProc:
		fields, err := semExprs(pctx.TypeContext, scope, p.Fields)
		if err != nil {
			return nil, fmt.Errorf("compiling top: %w", err)
		}
		return &kernel.Top{
			Op:     "Top",
			Fields: fields,
			Limit:  p.Limit,
			Flush:  p.Flush,
		}, nil

	case *ast.PutProc:
		assignments, err := semAssignments(pctx.TypeContext, scope, p.Clauses)
		if err != nil {
			return nil, err
		}
		return &kernel.Put{
			Op:          "Put",
			Assignments: assignments,
		}, nil

	case *ast.RenameProc:
		var assignments []kernel.FieldAssignment
		for _, fa := range p.Fields {
			dst, err := semField(fa.LHS)
			if err != nil {
				return nil, err
			}
			// We call CompileLval on the RHS because renames are
			// restricted to dotted field name expressions.
			src, err := semField(fa.RHS)
			if err != nil {
				return nil, err
			}
			if len(dst) != len(src) {
				return nil, fmt.Errorf("cannot rename %s to %s", src, dst)
			}
			//XXX kernel has to check this.
			// Check that the prefixes match and, if not, report first place
			// that they don't.
			for i := 0; i <= len(src)-2; i++ {
				if src[i] != dst[i] {
					return nil, fmt.Errorf("cannot rename %s to %s (differ in %s vs %s)", src, dst, src[i], dst[i])
				}
			}
			assignment := kernel.FieldAssignment{dst, src}
			assignments = append(assignments, assignment)
		}
		return &kernel.Rename{
			Op:          "Rename",
			Assignments: assignments,
		}, nil

	case *ast.FuseProc:
		return &kernel.Fuse{"Fuse"}, nil

	case *ast.FunctionCall:
		return nil, errors.New("internal error: semantic analyzer should have converted function in proc context to filter or group-by")

	case *ast.JoinProc:
		return nil, ErrJoinParents

	default:
		return nil, fmt.Errorf("unknown AST type: %v", p)

	}
}

func compileAssignments(assignments []ast.Assignment, zctx *resolver.Context, scope *Scope) ([]expr.Assignment, error) {
	keys := make([]expr.Assignment, 0, len(assignments))
	for _, assignment := range assignments {
		a, err := CompileAssignment(zctx, scope, &assignment)
		if err != nil {
			return nil, err
		}
		keys = append(keys, a)
	}
	return keys, nil
}

func splitAssignments(assignments []expr.Assignment) ([]field.Static, []expr.Evaluator) {
	n := len(assignments)
	lhs := make([]field.Static, 0, n)
	rhs := make([]expr.Evaluator, 0, n)
	for _, a := range assignments {
		lhs = append(lhs, a.LHS)
		rhs = append(rhs, a.RHS)
	}
	return lhs, rhs
}

func enteringJoin(nodes []ast.Proc) bool {
	var ok bool
	if len(nodes) > 0 {
		_, ok = nodes[0].(*ast.JoinProc)
	}
	return ok
}

func compileSequential(custom Hook, nodes []ast.Proc, pctx *proc.Context, scope *Scope, parents []proc.Interface) ([]proc.Interface, error) {
	node := nodes[0]
	parents, err := Compile(custom, node, pctx, scope, parents)
	if err != nil {
		return nil, err
	}
	// merge unless we're at the end of the chain,
	// in which case the output layer will mux
	// into channels.
	if len(nodes) == 1 {
		return parents, nil
	}
	if len(parents) > 1 && !enteringJoin(nodes[1:]) {
		var parent proc.Interface
		p := node.(*ast.ParallelProc)
		if p.MergeOrderField != nil {
			cmp := zbuf.NewCompareFn(p.MergeOrderField, p.MergeOrderReverse)
			parent = merge.New(pctx, parents, cmp)
		} else {
			parent = combine.New(pctx, parents)
		}
		parents = []proc.Interface{parent}
	}
	return compileSequential(custom, nodes[1:], pctx, scope, parents)
}

func compileParallel(custom Hook, pp *ast.ParallelProc, c *proc.Context, scope *Scope, parents []proc.Interface) ([]proc.Interface, error) {
	n := len(pp.Procs)
	if len(parents) == 1 {
		// Single parent: insert a splitter and wire to each branch.
		splitter := split.New(parents[0])
		parents = []proc.Interface{}
		for k := 0; k < n; k++ {
			sc := splitter.NewProc()
			parents = append(parents, sc)
		}
	}
	if len(parents) != n {
		return nil, fmt.Errorf("proc.CompileProc: %d parents for parallel proc with %d branches", len(parents), len(pp.Procs))
	}
	var procs []proc.Interface
	for k := 0; k < n; k++ {
		proc, err := Compile(custom, pp.Procs[k], c, scope, []proc.Interface{parents[k]})
		if err != nil {
			return nil, err
		}
		procs = append(procs, proc...)
	}
	return procs, nil
}

// Compile compiles an AST into a graph of Procs, and returns
// the leaves.  A custom compiler hook can be included and it will be tried first
// for each node encountered during the compilation.
func Compile(custom Hook, node ast.Proc, pctx *proc.Context, scope *Scope, parents []proc.Interface) ([]proc.Interface, error) {
	if len(parents) == 0 {
		return nil, errors.New("no parents")
	}
	if scope == nil {
		// Outermost caller should pass in global scope object.  If nil,
		// we assume no global context and allocate a fresh, empty scope.
		scope = newScope()
	}
	switch node := node.(type) {
	case *ast.SequentialProc:
		if len(node.Procs) == 0 {
			return nil, errors.New("sequential proc without procs")
		}
		return compileSequential(custom, node.Procs, pctx, scope, parents)

	case *ast.ParallelProc:
		return compileParallel(custom, node, pctx, scope, parents)

	case *ast.JoinProc:
		if len(parents) != 2 {
			return nil, ErrJoinParents
		}
		assignments, err := compileAssignments(node.Clauses, pctx.TypeContext, scope)
		if err != nil {
			return nil, err
		}
		lhs, rhs := splitAssignments(assignments)
		leftKey, err := compileExpr(pctx.TypeContext, scope, node.LeftKey)
		if err != nil {
			return nil, err
		}
		rightKey, err := compileExpr(pctx.TypeContext, scope, node.RightKey)
		if err != nil {
			return nil, err
		}
		join, err := join.New(pctx, parents[0], parents[1], leftKey, rightKey, lhs, rhs)
		if err != nil {
			return nil, err
		}
		return []proc.Interface{join}, nil

	default:
		if len(parents) > 1 {
			return nil, fmt.Errorf("ast type %v cannot have multiple parents", node)
		}
		p, err := compileProc(custom, node, pctx, scope, parents[0])
		return []proc.Interface{p}, err
	}
}

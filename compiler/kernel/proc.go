package kernel

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/compiler/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/proc/filter"
	"github.com/brimsec/zq/proc/fuse"
	"github.com/brimsec/zq/proc/head"
	"github.com/brimsec/zq/proc/join"
	"github.com/brimsec/zq/proc/pass"
	"github.com/brimsec/zq/proc/put"
	"github.com/brimsec/zq/proc/rename"
	"github.com/brimsec/zq/proc/sort"
	"github.com/brimsec/zq/proc/split"
	"github.com/brimsec/zq/proc/tail"
	"github.com/brimsec/zq/proc/top"
	"github.com/brimsec/zq/proc/uniq"
	"github.com/brimsec/zq/zng/resolver"
)

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

func compileOperator(custom Hook, o Operator, pctx *proc.Context, scope *Scope, parent proc.Interface) (proc.Interface, error) {
	if custom != nil {
		panic("get rid of custom procs")
		// XXX custom should take scope... just move custom procs
		// into the core...
		//p, err := custom(node, pctx, parent)
		//if err != nil {
		//	return nil, err
		//}
		//if p != nil {
		//	return p, err
		//}
	}
	switch o := o.(type) {
	case *Agg:
		return compileAgg(pctx, scope, parent, o)

	case *Cut:
		assignments, err := compileAssignments(o.Assignments, pctx.TypeContext, scope)
		if err != nil {
			return nil, err
		}
		lhs, rhs := splitAssignments(assignments)
		cutter, err := expr.NewCutter(pctx.TypeContext, lhs, rhs)
		if err != nil {
			return nil, err
		}
		cutter.AllowPartialCuts()
		return proc.FromFunction(pctx, parent, cutter, "cut"), nil

	case *Pick:
		assignments, err := compileAssignments(o.Assignments, pctx.TypeContext, scope)
		if err != nil {
			return nil, err
		}
		lhs, rhs := splitAssignments(assignments)
		cutter, err := expr.NewCutter(pctx.TypeContext, lhs, rhs)
		if err != nil {
			return nil, err
		}
		return proc.FromFunction(pctx, parent, cutter, "pick"), nil

	case *Drop:
		if len(o.Fields) == 0 {
			return nil, errors.New("drop: no fields given")
		}
		dropper := expr.NewDropper(pctx.TypeContext, o.Fields)
		return proc.FromFunction(pctx, parent, dropper, "drop"), nil

	case *Sort:
		fields, err := CompileExprs(pctx.TypeContext, scope, o.Fields)
		if err != nil {
			return nil, err
		}
		sortdir := 1
		if o.Reverse {
			sortdir = -1
		}
		sort, err := sort.New(pctx, parent, fields, sortdir, o.NullsFirst)
		if err != nil {
			return nil, fmt.Errorf("compiling sort: %w", err)
		}
		return sort, nil

	case *Head:
		limit := o.Count
		if limit == 0 {
			limit = 1
		}
		return head.New(parent, limit), nil

	case *Tail:
		limit := o.Count
		if limit == 0 {
			limit = 1
		}
		return tail.New(parent, limit), nil

	case *Uniq:
		return uniq.New(pctx, parent, o.Cflag), nil

	case *Pass:
		// analyzer should take this out?
		return pass.New(parent), nil

	case *Filter:
		f, err := compileFilter(pctx.TypeContext, scope, o.Predicate)
		if err != nil {
			return nil, fmt.Errorf("compiling filter: %w", err)
		}
		return filter.New(parent, f), nil

	case *Top:
		fields, err := CompileExprs(pctx.TypeContext, scope, o.Fields)
		if err != nil {
			return nil, fmt.Errorf("compiling top: %w", err)
		}
		return top.New(parent, o.Limit, fields, o.Flush), nil

	case *Put:
		clauses, err := compileAssignments(o.Assignments, pctx.TypeContext, scope)
		if err != nil {
			return nil, err
		}
		put, err := put.New(pctx, parent, clauses)
		if err != nil {
			return nil, err
		}
		return put, nil

	case *Rename:
		var dsts, srcs []field.Static
		for _, fa := range o.Assignments {
			// Check that the prefixes match and, if not, report first place
			// that they don't.
			src := fa.RHS
			dst := fa.LHS
			for i := 0; i <= len(src)-2; i++ {
				if src[i] != dst[i] {
					return nil, fmt.Errorf("cannot rename %s to %s (differ in %s vs %s)", src, dst, src[i], dst[i])
				}
			}
			dsts = append(dsts, dst)
			srcs = append(srcs, src)
		}
		renamer := rename.NewFunction(pctx.TypeContext, srcs, dsts)
		return proc.FromFunction(pctx, parent, renamer, "rename"), nil

	case *Fuse:
		return fuse.New(pctx, parent)

	case *Join:
		return nil, ErrJoinParents

	default:
		return nil, fmt.Errorf("unknown AST type: %v", o)

	}
}

func compileAssignments(assignments []Assignment, zctx *resolver.Context, scope *Scope) ([]expr.Assignment, error) {
	keys := make([]expr.Assignment, 0, len(assignments))
	for _, assignment := range assignments {
		a, err := CompileAssignment(zctx, scope, assignment)
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

func enteringJoin(ops []Operator) bool {
	var ok bool
	if len(ops) > 0 {
		_, ok = ops[0].(*Join)
	}
	return ok
}

func compileSequential(custom Hook, ops []Operator, pctx *proc.Context, scope *Scope, parents []proc.Interface) ([]proc.Interface, error) {
	o := ops[0]
	parents, err := Compile(custom, o, pctx, scope, parents)
	if err != nil {
		return nil, err
	}
	// merge unless we're at the end of the chain,
	// in which case the output layer will mux
	// into channels.
	if len(ops) == 1 {
		return parents, nil
	}

	/*
		//XXX semantic pass will do this restructure
		if len(parents) > 1 && !enteringJoin(ops[1:]) {
			var parent proc.Interface
			p := o.(*Parallel)
			if p.MergeOrderField != nil {
				cmp := zbuf.NewCompareFn(p.MergeOrderField, p.MergeOrderReverse)
				parent = merge.New(pctx, parents, cmp)
			} else {
				parent = combine.New(pctx, parents)
			}
			parents = []proc.Interface{parent}
		}
	*/

	return compileSequential(custom, ops[1:], pctx, scope, parents)
}

func compileParallel(custom Hook, par *Parallel, c *proc.Context, scope *Scope, parents []proc.Interface) ([]proc.Interface, error) {
	n := len(par.Operators)
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
		return nil, fmt.Errorf("proc.CompileProc: %d parents for parallel proc with %d branches", len(parents), len(par.Operators))
	}
	var procs []proc.Interface
	for k := 0; k < n; k++ {
		proc, err := Compile(custom, par.Operators[k], c, scope, []proc.Interface{parents[k]})
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
func Compile(custom Hook, o Operator, pctx *proc.Context, scope *Scope, parents []proc.Interface) ([]proc.Interface, error) {
	if len(parents) == 0 {
		return nil, errors.New("no parents")
	}
	if scope == nil {
		// Outermost caller should pass in global scope object.  If nil,
		// we assume no global context and allocate a fresh, empty scope.
		scope = newScope()
	}
	switch o := o.(type) {
	case *Sequential:
		if len(o.Operators) == 0 {
			return nil, errors.New("Z sequential without operators")
		}
		return compileSequential(custom, o.Operators, pctx, scope, parents)

	case *Parallel:
		return compileParallel(custom, o, pctx, scope, parents)

	case *Join:
		if len(parents) != 2 {
			return nil, ErrJoinParents
		}
		assignments, err := compileAssignments(o.Assignments, pctx.TypeContext, scope)
		if err != nil {
			return nil, err
		}
		lhs, rhs := splitAssignments(assignments)
		leftKey, err := compileExpr(pctx.TypeContext, scope, o.LeftKey)
		if err != nil {
			return nil, err
		}
		rightKey, err := compileExpr(pctx.TypeContext, scope, o.RightKey)
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
			return nil, fmt.Errorf("ast type %v cannot have multiple parents", o)
		}
		p, err := compileOperator(custom, o, pctx, scope, parents[0])
		return []proc.Interface{p}, err
	}
}

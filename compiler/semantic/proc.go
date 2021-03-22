package semantic

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/compiler/ast"
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/pkg/nano"
)

// semProc does a semantic analysis on a flowgraph to an
// intermediate representation that can be compiled into the runtime
// object.  Currently, it only replaces the group-by duration with
// a truncation call on the ts and replaces FunctionCall's in proc context
// with either a group-by or filter-proc based on the function's name.
func semProc(scope *Scope, p ast.Proc) (ast.Proc, error) {
	switch p := p.(type) {
	case *ast.Summarize:
		keys := p.Keys
		if duration := p.Duration; duration != nil {
			d, err := nano.ParseDuration(duration.Text)
			if err != nil {
				return nil, err
			}
			durationKey := ast.Assignment{
				Kind: "Assignment",
				LHS:  ast.NewDotExpr(field.New("ts")),
				RHS: &ast.Call{
					Kind: "Call",
					Name: "trunc",
					Args: []ast.Expr{
						ast.NewDotExpr(field.New("ts")),
						&ast.Primitive{
							Kind: "Primitive",
							Type: "duration",
							Text: d.String(),
						}},
				},
			}
			keys = append([]ast.Assignment{durationKey}, keys...)
		}
		var err error
		keys, err = semAssignments(scope, keys)
		if err != nil {
			return nil, err
		}
		aggs, err := semAssignments(scope, p.Aggs)
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
		// separation... see issue #2163.  Also, we copy Duration even though
		// above we changed duration to the a trunc(ts) group-by key as the
		// Duration field is used later by the parallelization operator.
		return &ast.Summarize{
			Kind:         "Summarize",
			Duration:     p.Duration,
			InputSortDir: p.InputSortDir,
			Limit:        p.Limit,
			Keys:         keys,
			Aggs:         aggs,
			PartialsIn:   p.PartialsIn,
			PartialsOut:  p.PartialsOut,
		}, nil
	case *ast.Parallel:
		var procs []ast.Proc
		for _, p := range p.Procs {
			if isConst(p) {
				continue
			}
			converted, err := semProc(scope, p)
			if err != nil {
				return nil, err
			}
			procs = append(procs, converted)
		}
		return &ast.Parallel{
			Kind:         "Parallel",
			MergeBy:      p.MergeBy,
			MergeReverse: p.MergeReverse,
			Procs:        procs,
		}, nil
	case *ast.Sequential:
		var procs []ast.Proc
		for _, p := range p.Procs {
			if isConst(p) {
				continue
			}
			converted, err := semProc(scope, p)
			if err != nil {
				return nil, err
			}
			procs = append(procs, converted)
		}
		return &ast.Sequential{
			Kind:  "Sequential",
			Procs: procs,
		}, nil
	case *ast.Switch:
		var cases []ast.Case
		for k := range p.Cases {
			var err error
			tr, err := semProc(scope, p.Cases[k].Proc)
			if err != nil {
				return nil, err
			}
			e, err := semExpr(scope, p.Cases[k].Expr)
			if err != nil {
				return nil, err
			}
			cases = append(cases, ast.Case{Expr: e, Proc: tr})
		}
		return &ast.Switch{
			Kind:         "Switch",
			Cases:        cases,
			MergeBy:      p.MergeBy,
			MergeReverse: p.MergeReverse,
		}, nil
	case *ast.Call:
		converted, err := convertFunctionProc(p)
		if err != nil {
			return nil, err
		}
		return semProc(scope, converted)
	case *ast.Shape:
		return p, nil
	case *ast.Cut:
		assignments, err := semAssignments(scope, p.Args)
		if err != nil {
			return nil, err
		}
		return &ast.Cut{
			Kind: "Cut",
			Args: assignments,
		}, nil
	case *ast.Pick:
		assignments, err := semAssignments(scope, p.Args)
		if err != nil {
			return nil, err
		}
		return &ast.Pick{
			Kind: "Pick",
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
		return &ast.Drop{
			Kind: "Drop",
			Args: args,
		}, nil
	case *ast.Sort:
		exprs, err := semExprs(scope, p.Args)
		if err != nil {
			return nil, fmt.Errorf("sort: %w", err)
		}
		return &ast.Sort{
			Kind:       "Sort",
			Args:       exprs,
			SortDir:    p.SortDir,
			NullsFirst: p.NullsFirst,
		}, nil
	case *ast.Head:
		limit := p.Count
		if limit == 0 {
			limit = 1
		}
		return &ast.Head{
			Kind:  "Head",
			Count: limit,
		}, nil
	case *ast.Tail:
		limit := p.Count
		if limit == 0 {
			limit = 1
		}
		return &ast.Tail{
			Kind:  "Tail",
			Count: limit,
		}, nil
	case *ast.Uniq:
		return &ast.Uniq{
			Kind:  "Uniq",
			Cflag: p.Cflag,
		}, nil
	case *ast.Pass:
		return p, nil
	case *ast.Filter:
		e, err := semExpr(scope, p.Expr)
		if err != nil {
			return nil, err
		}
		return &ast.Filter{
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
		return &ast.Top{
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
		return &ast.Put{
			Kind: "Put",
			Args: assignments,
		}, nil
	case *ast.Rename:
		var assignments []ast.Assignment
		for _, fa := range p.Args {
			dst, err := semField(scope, fa.LHS)
			if err != nil {
				return nil, err
			}
			src, err := semField(scope, fa.RHS)
			if err != nil {
				return nil, err
			}
			dstField, ok := dst.(*ast.Path)
			if !ok {
				return nil, errors.New("'rename' requires explicit field references")
			}
			srcField, ok := src.(*ast.Path)
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
			assignments = append(assignments, ast.Assignment{"Assignment", dst, src})
		}
		return &ast.Rename{
			Kind: "Rename",
			Args: assignments,
		}, nil
	case *ast.Fuse:
		return p, nil
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
		return &ast.Join{
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
			return nil, errors.New("unable to covert SQL expression to Z")
		}
		return semProc(scope, converted)

	case *ast.FieldCutter, *ast.TypeSplitter:
		return p, nil
	}
	return nil, fmt.Errorf("semantic transform: unknown AST type: %v", p)
}

func semConsts(consts []ast.Proc, scope *Scope, p ast.Proc) ([]ast.Proc, error) {
	switch p := p.(type) {
	case *ast.Sequential:
		for _, p := range p.Procs {
			var err error
			consts, err = semConsts(consts, scope, p)
			if err != nil {
				return nil, err
			}
		}
	case *ast.Parallel:
		for _, p := range p.Procs {
			var err error
			consts, err = semConsts(consts, scope, p)
			if err != nil {
				return nil, err
			}
		}
	case *ast.TypeProc:
		typ, err := semType(scope, p.Type)
		if err != nil {
			return nil, err
		}
		converted := &ast.TypeProc{
			Kind: "TypeProc",
			Name: p.Name,
			Type: typ,
		}
		scope.Bind(p.Name, converted)
		return append(consts, converted), nil
	case *ast.Const:
		e, err := semExpr(scope, p.Expr)
		if err != nil {
			return nil, err
		}
		converted := &ast.Const{
			Kind: "Const",
			Name: p.Name,
			Expr: e,
		}
		scope.Bind(p.Name, converted)
		return append(consts, converted), nil
	}
	return consts, nil
}

func isConst(p ast.Proc) bool {
	switch p.(type) {
	case *ast.Const, *ast.TypeProc:
		return true
	}
	return false
}

package semantic

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/ast/zed"
	"github.com/brimdata/zed/compiler/kernel"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/proc"
	"github.com/segmentio/ksuid"
)

func semFrom(ctx context.Context, scope *Scope, from *ast.From, adaptor proc.DataAdaptor) (*dag.From, error) {
	var trunks []dag.Trunk
	for _, in := range from.Trunks {
		converted, err := semTrunk(ctx, scope, in, adaptor)
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

func semTrunk(ctx context.Context, scope *Scope, trunk ast.Trunk, adaptor proc.DataAdaptor) (dag.Trunk, error) {
	source, err := semSource(ctx, scope, trunk.Source, adaptor)
	if err != nil {
		return dag.Trunk{}, err
	}
	seq, err := semSequential(ctx, scope, trunk.Seq, adaptor)
	if err != nil {
		return dag.Trunk{}, err
	}
	return dag.Trunk{
		Kind:   "Trunk",
		Source: source,
		Seq:    seq,
	}, nil
}

func semSource(ctx context.Context, scope *Scope, source ast.Source, adaptor proc.DataAdaptor) (dag.Source, error) {
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
		id, err := ParseID(p.Name)
		if err != nil {
			id, err = adaptor.Lookup(ctx, p.Name)
			if err != nil {
				return nil, err
			}
		}
		var at ksuid.KSUID
		if p.At != "" {
			at, err = ParseID(p.At)
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
		return &dag.Pool{
			Kind:      "Pool",
			ID:        id,
			ScanLower: lower,
			ScanUpper: upper,
			ScanOrder: p.ScanOrder,
			At:        at,
		}, nil
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

func semSequential(ctx context.Context, scope *Scope, seq *ast.Sequential, adaptor proc.DataAdaptor) (*dag.Sequential, error) {
	if seq == nil {
		return nil, nil
	}
	var ops []dag.Op
	for _, p := range seq.Procs {
		if isConst(p) {
			continue
		}
		converted, err := semProc(ctx, scope, p, adaptor)
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

// semProc does a semantic analysis on a flowgraph to an
// intermediate representation that can be compiled into the runtime
// object.  Currently, it only replaces the group-by duration with
// a truncation call on the ts and replaces FunctionCall's in proc context
// with either a group-by or filter-proc based on the function's name.
func semProc(ctx context.Context, scope *Scope, p ast.Proc, adaptor proc.DataAdaptor) (dag.Op, error) {
	switch p := p.(type) {
	case *ast.From:
		return semFrom(ctx, scope, p, adaptor)
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
						&zed.Primitive{
							Kind: "Primitive",
							Type: "duration",
							Text: d.String(),
						}},
				},
			}
			keys = append([]dag.Assignment{durationKey}, keys...)
		}
		aggs, err := semAssignments(scope, p.Aggs)
		if err != nil {
			return nil, err
		}
		var dur *zed.Primitive
		if p.Duration != nil {
			dur = &zed.Primitive{
				Kind: "Primitive",
				Type: p.Duration.Type,
				Text: p.Duration.Text,
			}
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
			if isConst(p) {
				continue
			}
			converted, err := semProc(ctx, scope, p, adaptor)
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
		return semSequential(ctx, scope, p, adaptor)
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
				// A nil expression indicates the default case.
				e = &zed.Primitive{
					Kind: "Primitive",
					Type: "bool",
					Text: "true",
				}
			}
			op, err := semProc(ctx, scope, c.Proc, adaptor)
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
		return convertFunctionProc(scope, p)
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
	case *ast.Pick:
		assignments, err := semAssignments(scope, p.Args)
		if err != nil {
			return nil, err
		}
		return &dag.Pick{
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
	}
	return nil, fmt.Errorf("semantic transform: unknown AST type: %v", p)
}

func semConsts(consts []dag.Op, scope *Scope, p ast.Proc) ([]dag.Op, error) {
	switch p := p.(type) {
	case *ast.From:
		for _, trunk := range p.Trunks {
			if trunk.Seq == nil {
				continue
			}
			var err error
			consts, err = semConsts(consts, scope, trunk.Seq)
			if err != nil {
				return nil, err
			}
		}
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
		converted := &dag.TypeProc{
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
		converted := &dag.Const{
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

//XXX this needs to find a common home that doesn't import lake
func ParseID(s string) (ksuid.KSUID, error) {
	// Check if this is a cut-and-paste from ZNG, which encodes
	// the 20-byte KSUID as a 40 character hex string with 0x prefix.
	var id ksuid.KSUID
	if len(s) == 42 && s[0:2] == "0x" {
		b, err := hex.DecodeString(s[2:])
		if err != nil {
			return ksuid.Nil, fmt.Errorf("illegal hex tag: %s", s)
		}
		id, err = ksuid.FromBytes(b)
		if err != nil {
			return ksuid.Nil, fmt.Errorf("illegal hex tag: %s", s)
		}
	} else {
		var err error
		id, err = ksuid.Parse(s)
		if err != nil {
			return ksuid.Nil, fmt.Errorf("%s: invalid commit ID", s)
		}
	}
	return id, nil
}

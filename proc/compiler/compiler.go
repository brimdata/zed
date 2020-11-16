package compiler

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/filter"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/proc/combine"
	"github.com/brimsec/zq/proc/cut"
	filterproc "github.com/brimsec/zq/proc/filter"
	"github.com/brimsec/zq/proc/fuse"
	"github.com/brimsec/zq/proc/groupby"
	"github.com/brimsec/zq/proc/head"
	"github.com/brimsec/zq/proc/merge"
	"github.com/brimsec/zq/proc/pass"
	"github.com/brimsec/zq/proc/put"
	"github.com/brimsec/zq/proc/rename"
	"github.com/brimsec/zq/proc/sort"
	"github.com/brimsec/zq/proc/split"
	"github.com/brimsec/zq/proc/tail"
	"github.com/brimsec/zq/proc/top"
	"github.com/brimsec/zq/proc/uniq"
	"github.com/brimsec/zq/zbuf"
)

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

func compileProc(custom Hook, node ast.Proc, pctx *proc.Context, parent proc.Interface) (proc.Interface, error) {
	if custom != nil {
		p, err := custom(node, pctx, parent)
		if err != nil {
			return nil, err
		}
		if p != nil {
			return p, err
		}
	}
	switch v := node.(type) {
	case *ast.GroupByProc:
		return groupby.New(pctx, parent, v)

	case *ast.CutProc:
		cut, err := cut.New(pctx, parent, v)
		if err != nil {
			return nil, err
		}
		return cut, nil

	case *ast.SortProc:
		sort, err := sort.New(pctx, parent, v)
		if err != nil {
			return nil, fmt.Errorf("compiling sort: %w", err)
		}
		return sort, nil

	case *ast.HeadProc:
		limit := v.Count
		if limit == 0 {
			limit = 1
		}
		return head.New(parent, limit), nil

	case *ast.TailProc:
		limit := v.Count
		if limit == 0 {
			limit = 1
		}
		return tail.New(parent, limit), nil

	case *ast.UniqProc:
		return uniq.New(pctx, parent, v.Cflag), nil

	case *ast.PassProc:
		return pass.New(parent), nil

	case *ast.FilterProc:
		f, err := filter.Compile(pctx.TypeContext, v.Filter)
		if err != nil {
			return nil, fmt.Errorf("compiling filter: %w", err)
		}
		return filterproc.New(parent, f), nil

	case *ast.TopProc:
		fields, err := expr.CompileExprs(pctx.TypeContext, v.Fields)
		if err != nil {
			return nil, fmt.Errorf("compiling top: %w", err)
		}
		return top.New(parent, v.Limit, fields, v.Flush), nil

	case *ast.PutProc:
		put, err := put.New(pctx, parent, v)
		if err != nil {
			return nil, err
		}
		return put, nil

	case *ast.RenameProc:
		rename, err := rename.New(pctx, parent, v)
		if err != nil {
			return nil, err
		}
		return rename, nil

	case *ast.FuseProc:
		return fuse.New(pctx, parent)

	default:
		return nil, fmt.Errorf("unknown AST type: %v", v)

	}
}

func compileSequential(custom Hook, nodes []ast.Proc, pctx *proc.Context, parents []proc.Interface) ([]proc.Interface, error) {
	node := nodes[0]
	parents, err := Compile(custom, node, pctx, parents)
	if err != nil {
		return nil, err
	}
	// merge unless we're at the end of the chain,
	// in which case the output layer will mux
	// into channels.
	if len(nodes) == 1 {
		return parents, nil
	}
	if len(parents) > 1 {
		var parent proc.Interface
		p := node.(*ast.ParallelProc)
		if p.MergeOrderField != nil {
			cmp := zbuf.NewCompareFn(p.MergeOrderField, p.MergeOrderReverse)
			parent = merge.New(pctx, parents, cmp)
		} else {
			parent = combine.New(pctx, parents)
		}
		parents = parents[0:1]
		parents[0] = parent
	}
	return compileSequential(custom, nodes[1:], pctx, parents)
}

func compileParallel(custom Hook, pp *ast.ParallelProc, c *proc.Context, parents []proc.Interface) ([]proc.Interface, error) {
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
		proc, err := Compile(custom, pp.Procs[k], c, []proc.Interface{parents[k]})
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
func Compile(custom Hook, node ast.Proc, pctx *proc.Context, parents []proc.Interface) ([]proc.Interface, error) {
	if len(parents) == 0 {
		return nil, errors.New("no parents")
	}
	switch node := node.(type) {
	case *ast.SequentialProc:
		if len(node.Procs) == 0 {
			return nil, errors.New("sequential proc without procs")
		}
		return compileSequential(custom, node.Procs, pctx, parents)

	case *ast.ParallelProc:
		return compileParallel(custom, node, pctx, parents)

	default:
		if len(parents) > 1 {
			return nil, fmt.Errorf("ast type %v cannot have multiple parents", node)
		}
		p, err := compileProc(custom, node, pctx, parents[0])
		return []proc.Interface{p}, err
	}
}

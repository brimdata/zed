package compiler

import (
	"fmt"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/filter"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/proc/cut"
	filterproc "github.com/brimsec/zq/proc/filter"
	"github.com/brimsec/zq/proc/groupby"
	"github.com/brimsec/zq/proc/head"
	"github.com/brimsec/zq/proc/merge"
	"github.com/brimsec/zq/proc/orderedmerge"
	"github.com/brimsec/zq/proc/pass"
	"github.com/brimsec/zq/proc/put"
	"github.com/brimsec/zq/proc/rename"
	"github.com/brimsec/zq/proc/sort"
	"github.com/brimsec/zq/proc/split"
	"github.com/brimsec/zq/proc/tail"
	"github.com/brimsec/zq/proc/top"
	"github.com/brimsec/zq/proc/uniq"
)

//XXX this should go away
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

// Compile compiles an AST into a graph of Procs, and returns
// the leaves.  A custom proc compiler can be included and it will be tried first
// for each node encountered during the compilation.
func Compile(custom Hook, node ast.Proc, ctx *proc.Context, parents []proc.Interface) ([]proc.Interface, error) {
	if !isContainerProc(node) && len(parents) != 1 {
		return nil, fmt.Errorf("proc.CompileProc: expected single parent for node %T, got %d", node, len(parents))
	}
	parent := parents[0]

	if custom != nil {
		p, err := custom(node, ctx, parent)
		if err != nil {
			return nil, err
		}
		if p != nil {
			return []proc.Interface{p}, err
		}
	}
	switch v := node.(type) {

	case *ast.GroupByProc:
		params, err := groupby.CompileParams(v, ctx.TypeContext)
		if err != nil {
			return nil, err
		}
		return []proc.Interface{groupby.New(ctx, parent, *params)}, nil

	case *ast.CutProc:
		cut, err := cut.New(ctx, parent, v)
		if err != nil {
			return nil, err
		}
		return []proc.Interface{cut}, nil

	case *ast.SortProc:
		sort, err := sort.New(ctx, parent, v)
		if err != nil {
			return nil, fmt.Errorf("compiling sort: %w", err)
		}
		return []proc.Interface{sort}, nil

	case *ast.HeadProc:
		limit := v.Count
		if limit == 0 {
			limit = 1
		}
		return []proc.Interface{head.New(parent, limit)}, nil

	case *ast.TailProc:
		limit := v.Count
		if limit == 0 {
			limit = 1
		}
		return []proc.Interface{tail.New(parent, limit)}, nil

	case *ast.UniqProc:
		return []proc.Interface{uniq.New(ctx, parent, v.Cflag)}, nil

	case *ast.PassProc:
		return []proc.Interface{pass.New(parent)}, nil

	case *ast.FilterProc:
		f, err := filter.Compile(v.Filter)
		if err != nil {
			return nil, fmt.Errorf("compiling filter: %w", err)
		}
		return []proc.Interface{filterproc.New(parent, f)}, nil

	case *ast.TopProc:
		fields, err := expr.CompileFieldExprs(v.Fields)
		if err != nil {
			return nil, fmt.Errorf("compiling top: %w", err)
		}
		return []proc.Interface{top.New(parent, v.Limit, fields, v.Flush)}, nil

	case *ast.PutProc:
		put, err := put.New(ctx, parent, v)
		if err != nil {
			return nil, err
		}
		return []proc.Interface{put}, nil

	case *ast.RenameProc:
		rename, err := rename.New(ctx, parent, v)
		if err != nil {
			return nil, err
		}
		return []proc.Interface{rename}, nil

	case *ast.SequentialProc:
		var err error
		n := len(v.Procs)
		for k := 0; k < n; k++ {
			parents, err = Compile(custom, v.Procs[k], ctx, parents)
			if err != nil {
				return nil, err
			}
			// merge unless we're at the end of the chain,
			// in which case the output layer will mux
			// into channels.
			if len(parents) > 1 && k < n-1 {
				p := v.Procs[k].(*ast.ParallelProc)
				if p.MergeOrderField != "" {
					parents = []proc.Interface{orderedmerge.New(ctx, parents, p.MergeOrderField, p.MergeOrderReverse)}
				} else {
					parents = []proc.Interface{merge.New(ctx, parents)}
				}
				continue
			} else {
				parent = parents[0]
			}
		}
		return parents, nil

	case *ast.ParallelProc:
		return CompileParallel(custom, v, ctx, parents)

	default:
		return nil, fmt.Errorf("unknown AST type: %v", v)
	}
}

func CompileParallel(custom Hook, pp *ast.ParallelProc, c *proc.Context, parents []proc.Interface) ([]proc.Interface, error) {
	n := len(pp.Procs)
	if len(parents) == 1 {
		// Single parent: insert a splitter and wire to each branch.
		//XXX change split package to splitter
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

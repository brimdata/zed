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

type Custom interface {
	Compile(ast.Proc, proc.Parent) (proc.Interface, error)
}

// Compile compiles an AST into a graph of Procs, and returns
// the leaves.  A custom proc compiler can be included and it will be tried first
// for each node encountered during the compilation.
func Compile(custom Custom, node ast.Proc, parent proc.Parent) ([]proc.Interface, error) {
	if custom != nil {
		p, err := custom.Compile(node, parent)
		if err != nil {
			return nil, err
		}
		if p != nil {
			return []proc.Interface{p}, err
		}
	}
	switch v := node.(type) {

	case *ast.GroupByProc:
		params, err := groupby.CompileParams(v, parent.TypeContext)
		if err != nil {
			return nil, err
		}
		return []proc.Interface{groupby.New(parent, *params)}, nil

	case *ast.CutProc:
		cut, err := cut.New(parent, v)
		if err != nil {
			return nil, err
		}
		return []proc.Interface{cut}, nil

	case *ast.SortProc:
		sort, err := sort.New(parent, v)
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
		return []proc.Interface{uniq.New(parent, v.Cflag)}, nil

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
		put, err := put.New(parent, v)
		if err != nil {
			return nil, err
		}
		return []proc.Interface{put}, nil

	case *ast.RenameProc:
		rename, err := rename.New(parent, v)
		if err != nil {
			return nil, err
		}
		return []proc.Interface{rename}, nil

	case *ast.SequentialProc:
		var parents []proc.Interface
		var err error
		n := len(v.Procs)
		for k := 0; k < n; k++ {
			parents, err = Compile(custom, v.Procs[k], parent)
			if err != nil {
				return nil, err
			}
			// merge unless we're at the end of the chain,
			// in which case the output layer will mux
			// into channels.
			if len(parents) > 1 && k < n-1 {
				p := v.Procs[k].(*ast.ParallelProc)
				var mp proc.Interface
				if p.MergeOrderField != "" {
					mp = orderedmerge.New(parent.Context, parents, p.MergeOrderField, p.MergeOrderReverse)
				} else {
					mp = merge.New(parent.Context, parents)
				}
				parent = parent.Link(mp)
				continue
			}
			parent = parent.Link(parents[0])
		}
		return parents, nil

	case *ast.ParallelProc:
		splitter := split.New(parent)
		n := len(v.Procs)
		var procs []proc.Interface
		for k := 0; k < n; k++ {
			//
			// for each downstream proc chain, create a new SplitChannel,
			// attach the SplitChannel to the SplitProc, then generate the
			// proc chain with the SplitChannel as the new parent
			//
			sc := split.NewChannel(splitter)
			proc, err := Compile(custom, v.Procs[k], parent.Link(sc))
			if err != nil {
				return nil, err
			}
			procs = append(procs, proc...)
		}
		return procs, nil

	default:
		return nil, fmt.Errorf("unknown AST type: %v", v)
	}
}

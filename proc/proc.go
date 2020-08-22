package proc

import (
	"context"
	"fmt"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/filter"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng/resolver"
	"go.uber.org/zap"
)

// Proc is the interface to objects that operate on Batches of zbuf.Records
// and are arranged into a flowgraph to perform pattern matching and analytics.
// A proc is generally single-threaded unless lengths are taken to implement
// concurrency within a Proc.  The model is receiver-driven, stream-oriented
// data processing.  Downstream Procs Pull() batches of data from upstream procs.
// Normally, a Proc pulls data until end of stream (nil batch and nil error)
// or error (nil batch and non-nil error).  If a Proc wants to end before
// end of stream, it calls the Done() method on its parent.  A Proc implementation
// may assume calls to Pull() and Done() are single threaded so any arrangement
// of calls to Pull() and Done() cannot be done concurrently.  In short, never
// call Done() concurrently to another goroutine calling Pull()
type Proc interface {
	Pull() (zbuf.Batch, error)
	Done()
	Parents() []Proc
}

// Result is a convenient way to bundle the result of Proc.Pull() to
// send over channels.
type Result struct {
	Batch zbuf.Batch
	Err   error
}

// Context provides states used by all procs to provide the outside context
// in which they are running.
type Context struct {
	context.Context
	TypeContext *resolver.Context
	Logger      *zap.Logger
	Warnings    chan string
}

type Base struct {
	*Context
	Parent Proc
}

func EOS(batch zbuf.Batch, err error) bool {
	return batch == nil || err != nil
}

func (b *Base) Done() {
	if b.Parent != nil {
		b.Parent.Done()
	}
}

func (b *Base) Parents() []Proc {
	if b.Parent == nil {
		return []Proc{}
	}
	return []Proc{b.Parent}
}

func (b *Base) Get() (zbuf.Batch, error) {
	return b.Parent.Pull()
}

type Compiler interface {
	Compile(ast.Proc, *Context, Proc) (Proc, error)
}

// CompileProc compiles an AST into a graph of Procs, and returns
// the leaves.  A custom proc compiler can be included and it will be tried first
// for each node encountered during the compilation.
func CompileProc(custom Compiler, node ast.Proc, c *Context, parent Proc) ([]Proc, error) {
	if custom != nil {
		p, err := custom.Compile(node, c, parent)
		if err != nil {
			return nil, err
		}
		if p != nil {
			return []Proc{p}, err
		}
	}
	switch v := node.(type) {

	case *ast.GroupByProc:
		params, err := CompileGroupBy(v, c.TypeContext)
		if err != nil {
			return nil, err
		}
		return []Proc{NewGroupBy(c, parent, *params)}, nil

	case *ast.CutProc:
		cut, err := CompileCutProc(c, parent, v)
		if err != nil {
			return nil, err
		}
		return []Proc{cut}, nil

	case *ast.SortProc:
		sort, err := CompileSortProc(c, parent, v)
		if err != nil {
			return nil, fmt.Errorf("compiling sort: %w", err)
		}
		return []Proc{sort}, nil

	case *ast.HeadProc:
		limit := v.Count
		if limit == 0 {
			limit = 1
		}
		return []Proc{NewHead(c, parent, limit)}, nil

	case *ast.TailProc:
		limit := v.Count
		if limit == 0 {
			limit = 1
		}
		return []Proc{NewTail(c, parent, limit)}, nil

	case *ast.UniqProc:
		return []Proc{NewUniq(c, parent, v.Cflag)}, nil

	case *ast.PassProc:
		return []Proc{NewPass(c, parent)}, nil

	case *ast.FilterProc:
		f, err := filter.Compile(v.Filter)
		if err != nil {
			return nil, fmt.Errorf("compiling filter: %w", err)
		}
		return []Proc{NewFilter(c, parent, f)}, nil

	case *ast.TopProc:
		fields, err := expr.CompileFieldExprs(v.Fields)
		if err != nil {
			return nil, fmt.Errorf("compiling top: %w", err)
		}
		return []Proc{NewTop(c, parent, v.Limit, fields, v.Flush)}, nil

	case *ast.PutProc:
		put, err := CompilePutProc(c, parent, v)
		if err != nil {
			return nil, err
		}
		return []Proc{put}, nil

	case *ast.RenameProc:
		rename, err := CompileRenameProc(c, parent, v)
		if err != nil {
			return nil, err
		}
		return []Proc{rename}, nil

	case *ast.SequentialProc:
		var parents []Proc
		var err error
		n := len(v.Procs)
		for k := 0; k < n; k++ {
			parents, err = CompileProc(custom, v.Procs[k], c, parent)
			if err != nil {
				return nil, err
			}
			// merge unless we're at the end of the chain,
			// in which case the output layer will mux
			// into channels.
			if len(parents) > 1 && k < n-1 {
				p := v.Procs[k].(*ast.ParallelProc)
				if p.MergeOrderField != "" {
					parent = NewOrderedMerge(c, parents, p.MergeOrderField, p.MergeOrderReverse)
				} else {
					parent = NewMerge(c, parents)
				}
				continue
			}
			parent = parents[0]
		}
		return parents, nil

	case *ast.ParallelProc:
		splitter := NewSplit(c, parent)
		n := len(v.Procs)
		var procs []Proc
		for k := 0; k < n; k++ {
			//
			// for each downstream proc chain, create a new SplitChannel,
			// attach the SplitChannel to the SplitProc, then generate the
			// proc chain with the SplitChannel as the new parent
			//
			sc := NewSplitChannel(splitter)
			proc, err := CompileProc(custom, v.Procs[k], c, sc)
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

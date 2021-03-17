package semantic

import (
	"encoding/json"
	"fmt"

	"github.com/brimsec/zq/compiler/ast"
	"github.com/brimsec/zq/field"
)

var passProc = &ast.Pass{Kind: "Pass"}

//XXX
func zbufDirInt(reversed bool) int {
	if reversed {
		return -1
	}
	return 1
}

func ensureSequentialProc(p ast.Proc) *ast.Sequential {
	if p, ok := p.(*ast.Sequential); ok {
		return p
	}
	return &ast.Sequential{
		Kind:  "Sequential",
		Procs: []ast.Proc{p},
	}
}

func countConsts(procs []ast.Proc) int {
	for k, p := range procs {
		switch p.(type) {
		case *ast.Const, *ast.TypeProc:
			continue
		default:
			return k
		}
	}
	return 0
}

// liftFilter removes the filter at the head of the flowgraph AST, if
// one is present, and returns its ast.Expression and the modified
// flowgraph AST. If the flowgraph does not start with a filter, it
// returns nil and the unmodified flowgraph.
func liftFilter(p ast.Proc) (ast.Expr, ast.Proc) {
	if fp, ok := p.(*ast.Filter); ok {
		return fp.Expr, passProc
	}
	seq, ok := p.(*ast.Sequential)
	if ok && len(seq.Procs) > 0 {
		nc := countConsts(seq.Procs)
		if nc != 0 {
			panic("internal error: consts should have been removed from AST")
		}
		if fp, ok := seq.Procs[0].(*ast.Filter); ok {
			rest := ast.Proc(passProc)
			if len(seq.Procs) > 1 {
				rest = &ast.Sequential{
					Kind:  "Sequential",
					Procs: seq.Procs[1:],
				}
			}
			return fp.Expr, rest
		}
	}
	return nil, p
}

// all fields should be turned into field paths by initial semantic pass

func exprToField(e ast.Expr) field.Static {
	f, ok := e.(*ast.Path)
	if !ok {
		return nil
	}
	return f.Name
}

func eq(e ast.Expr, b field.Static) bool {
	a := exprToField(e)
	if a == nil {
		return false
	}
	return a.Equal(b)
}

// SetGroupByProcInputSortDir examines p under the assumption that its input is
// sorted according to inputSortField and inputSortDir.  If p is an
// ast.GroupByProc and SetGroupByProcInputSortDir can determine that its first
// grouping key is inputSortField or an order-preserving function of
// inputSortField, SetGroupByProcInputSortDir sets ast.GroupByProc.InputSortDir
// to inputSortDir.  SetGroupByProcInputSortDir returns true if it determines
// that p's output will remain sorted according to inputSortField and
// inputSortDir; otherwise, it returns false.
func SetGroupByProcInputSortDir(p ast.Proc, inputSortField field.Static, inputSortDir int) bool {
	switch p := p.(type) {
	case *ast.Cut:
		// Return true if the output record contains inputSortField.
		for _, f := range p.Args {
			if eq(f.RHS, inputSortField) {
				return true
			}
		}
		return false
	case *ast.Pick:
		// Return true if the output record contains inputSortField.
		for _, f := range p.Args {
			if eq(f.RHS, inputSortField) {
				return true
			}
		}
		return false
	case *ast.Drop:
		// Return true if the output record contains inputSortField.
		for _, e := range p.Args {
			if eq(e, inputSortField) {
				return false
			}
		}
		return true
	case *ast.Summarize:
		// Set p.InputSortDir and return true if the first grouping key
		// is inputSortField or an order-preserving function of it.
		if len(p.Keys) > 0 && eq(p.Keys[0].LHS, inputSortField) {
			rhs := exprToField(p.Keys[0].RHS)
			if rhs != nil && rhs.Equal(inputSortField) {
				p.InputSortDir = inputSortDir
				return true
			}
			if call, ok := p.Keys[0].RHS.(*ast.Call); ok {
				switch call.Name {
				case "ceil", "floor", "round", "trunc":
					if len(call.Args) == 0 {
						return false
					}
					arg0 := exprToField(call.Args[0])
					if arg0 != nil && arg0.Equal(inputSortField) {
						p.InputSortDir = inputSortDir
						return true
					}
				}
			}
		}
		return false
	case *ast.Put:
		for _, c := range p.Args {
			lhs := exprToField(c.LHS)
			if lhs != nil && lhs.Equal(inputSortField) {
				return false
			}
		}
		return true
	case *ast.Sequential:
		for _, pp := range p.Procs {
			if !SetGroupByProcInputSortDir(pp, inputSortField, inputSortDir) {
				return false
			}
		}
		return true
	case *ast.Filter, *ast.Head, *ast.Pass, *ast.Uniq, *ast.Tail, *ast.Fuse, *ast.Const, *ast.TypeProc:
		return true
	default:
		return false
	}
}

func copyProcs(ps []ast.Proc) []ast.Proc {
	var copies []ast.Proc
	for _, p := range ps {
		copies = append(copies, copyProc(p))
	}
	return copies
}

func copyProc(p ast.Proc) ast.Proc {
	if p == nil {
		panic("copyProc nil")
	}
	b, err := json.Marshal(p)
	if err != nil {
		panic(err)
	}
	copy, err := ast.UnpackJSONAsProc(b)
	if err != nil {
		panic(err)
	}
	return copy
}

func buildSplitFlowgraph(branch, tail []ast.Proc, mergeField field.Static, reverse bool, N int) (*ast.Sequential, bool) {
	if len(branch) == 0 {
		return &ast.Sequential{
			Kind:  "Sequential",
			Procs: tail,
		}, false
	}
	if len(tail) == 0 && mergeField != nil {
		// Insert a pass tail in order to force a merge of the
		// parallel branches when compiling. (Trailing parallel branches are wired to
		// a mux output).
		tail = []ast.Proc{passProc}
	}
	pp := &ast.Parallel{
		Kind:         "Parallel",
		Procs:        []ast.Proc{},
		MergeBy:      mergeField,
		MergeReverse: reverse,
	}
	for i := 0; i < N; i++ {
		pp.Procs = append(pp.Procs, &ast.Sequential{
			Kind:  "Sequential",
			Procs: copyProcs(branch),
		})
	}
	return &ast.Sequential{
		Kind:  "Sequential",
		Procs: append([]ast.Proc{pp}, tail...),
	}, true
}

func parallelize(p ast.Proc, N int, inputSortField field.Static, inputSortReversed bool) (*ast.Sequential, bool) {
	if p == nil {
		panic("parallelize nil")
	}

	seq := ensureSequentialProc(p)
	orderSensitiveTail := true
loop:
	for i := range seq.Procs {
		switch seq.Procs[i].(type) {
		case *ast.Sort, *ast.Summarize:
			orderSensitiveTail = false
			break loop
		default:
			continue
		}
	}
	for i := range seq.Procs {
		switch p := seq.Procs[i].(type) {
		case *ast.Filter, *ast.Pass, *ast.Const, *ast.TypeProc:
			// Stateless procs: continue until we reach one of the procs below at
			// which point we'll either split the flowgraph or see we can't and return it as-is.
			continue
		case *ast.Cut, *ast.Pick:
			if inputSortField == nil || !orderSensitiveTail {
				continue
			}
			var fields []ast.Assignment
			if cut, ok := p.(*ast.Cut); ok {
				fields = cut.Args
			} else {
				fields = p.(*ast.Pick).Args
			}
			var found bool
			for _, f := range fields {
				fieldName := exprToField(f.RHS)
				lhs := exprToField(f.LHS)
				if fieldName != nil && !fieldName.Equal(inputSortField) && lhs.Equal(inputSortField) {
					return buildSplitFlowgraph(seq.Procs[0:i], seq.Procs[i:], inputSortField, inputSortReversed, N)
				}
				if fieldName != nil && fieldName.Equal(inputSortField) && (lhs == nil || lhs.Equal(inputSortField)) {
					found = true
				}
			}
			if !found {
				return buildSplitFlowgraph(seq.Procs[0:i], seq.Procs[i:], inputSortField, inputSortReversed, N)
			}
		case *ast.Drop:
			if inputSortField == nil || !orderSensitiveTail {
				continue
			}
			for _, e := range p.Args {
				if eq(e, inputSortField) {
					return buildSplitFlowgraph(seq.Procs[0:i], seq.Procs[i:], inputSortField, inputSortReversed, N)
				}
			}
			continue
		case *ast.Put:
			if inputSortField == nil || !orderSensitiveTail {
				continue
			}
			for _, c := range p.Args {
				if eq(c.LHS, inputSortField) {
					return buildSplitFlowgraph(seq.Procs[0:i], seq.Procs[i:], inputSortField, inputSortReversed, N)
				}
			}
			continue
		case *ast.Rename:
			if inputSortField == nil || !orderSensitiveTail {
				continue
			}
			for _, f := range p.Args {
				if eq(f.LHS, inputSortField) {
					return buildSplitFlowgraph(seq.Procs[0:i], seq.Procs[i:], inputSortField, inputSortReversed, N)
				}
			}
		case *ast.Summarize:
			// To decompose the groupby, we split the flowgraph into branches that run up to and including a groupby,
			// followed by a post-merge groupby that composes the results.
			var mergeField field.Static
			if p.Duration.Seconds != 0 {
				// Group by time requires a time-ordered merge, irrespective of any upstream ordering.
				mergeField = field.New("ts")
			}
			branch := copyProcs(seq.Procs[0 : i+1])
			branch[len(branch)-1].(*ast.Summarize).PartialsOut = true

			composerGroupBy := copyProcs([]ast.Proc{p})[0].(*ast.Summarize)
			composerGroupBy.PartialsIn = true

			return buildSplitFlowgraph(branch, append([]ast.Proc{composerGroupBy}, seq.Procs[i+1:]...), mergeField, inputSortReversed, N)
		case *ast.Sort:
			dir := map[int]bool{-1: true, 1: false}[p.SortDir]
			if len(p.Args) == 1 {
				// Single sort field: we can sort in each parallel branch, and then do an ordered merge.
				mergeField := exprToField(p.Args[0])
				if mergeField == nil {
					return seq, false
				}
				return buildSplitFlowgraph(seq.Procs[0:i+1], seq.Procs[i+1:], mergeField, dir, N)
			} else {
				// Unknown or multiple sort fields: we sort after the merge point, which can be unordered.
				return buildSplitFlowgraph(seq.Procs[0:i], seq.Procs[i:], nil, dir, N)
			}
		case *ast.Parallel:
			return seq, false
		case *ast.Head, *ast.Tail:
			if inputSortField == nil {
				// Unknown order: we can't parallelize because we can't maintain this unknown order at the merge point.
				return seq, false
			}
			// put one head/tail on each parallel branch and one after the merge.
			return buildSplitFlowgraph(seq.Procs[0:i+1], seq.Procs[i:], inputSortField, inputSortReversed, N)
		case *ast.Uniq, *ast.Fuse:
			if inputSortField == nil {
				// Unknown order: we can't parallelize because we can't maintain this unknown order at the merge point.
				return seq, false
			}
			// Split all the upstream procs into parallel branches, then merge and continue with this and any remaining procs.
			return buildSplitFlowgraph(seq.Procs[0:i], seq.Procs[i:], inputSortField, inputSortReversed, N)
		case *ast.Sequential:
			return seq, false
			// XXX Joins can be parallelized but we need to write
			// the code to parallelize the flow graph, which is a bit
			// different from how group-bys are parallelized.
		case *ast.Join:
			return seq, false
		default:
			panic(fmt.Sprintf("proc type not handled: %T", p))
		}
	}
	// If we're here, we reached the end of the flowgraph without
	// coming across a merge-forcing proc. If inputs are sorted,
	// we can parallelize the entire chain and do an ordered
	// merge. Otherwise, no parallelization.
	if inputSortField == nil {
		return seq, false
	}
	return buildSplitFlowgraph(seq.Procs, nil, inputSortField, inputSortReversed, N)
}

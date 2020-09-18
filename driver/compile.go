package driver

import (
	"context"
	"encoding/json"
	"runtime"
	"strconv"
	"time"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/proc/compiler"
	"github.com/brimsec/zq/reducer"
	rcompile "github.com/brimsec/zq/reducer/compile"
	"github.com/brimsec/zq/zng/resolver"
	"go.uber.org/zap"
)

type Config struct {
	Custom            compiler.Hook
	Logger            *zap.Logger
	ReaderSortKey     string
	ReaderSortReverse bool
	Span              nano.Span
	StatsTick         <-chan time.Time
	Warnings          chan string
}

func compile(ctx context.Context, program ast.Proc, zctx *resolver.Context, msrc MultiSource, mcfg MultiConfig) (*muxOutput, error) {
	if mcfg.Logger == nil {
		mcfg.Logger = zap.NewNop()
	}
	if mcfg.Span.Dur == 0 {
		mcfg.Span = nano.MaxSpan
	}
	if mcfg.Warnings == nil {
		mcfg.Warnings = make(chan string, 5)
	}
	if mcfg.Parallelism == 0 {
		mcfg.Parallelism = runtime.GOMAXPROCS(0)
	}

	ReplaceGroupByProcDurationWithKey(program)

	sortKey, sortReversed := msrc.OrderInfo()
	if sortKey != "" {
		setGroupByProcInputSortDir(program, sortKey, zbufDirInt(sortReversed))
	}
	var filterExpr ast.BooleanExpr
	filterExpr, program = liftFilter(program)

	var isParallel bool
	if mcfg.Parallelism > 1 {
		program, isParallel = parallelizeFlowgraph(ensureSequentialProc(program), mcfg.Parallelism, sortKey, sortReversed)
	}
	if !isParallel {
		mcfg.Parallelism = 1
	}

	pctx := &proc.Context{
		Context:     ctx,
		TypeContext: zctx,
		Logger:      mcfg.Logger,
		Warnings:    mcfg.Warnings,
	}
	sources, pgroup, err := createParallelGroup(pctx, filterExpr, msrc, mcfg)
	if err != nil {
		return nil, err
	}

	leaves, err := compiler.Compile(mcfg.Custom, program, pctx, sources)
	if err != nil {
		return nil, err
	}
	return newMuxOutput(pctx, leaves, pgroup), nil
}

func ensureSequentialProc(p ast.Proc) *ast.SequentialProc {
	if p, ok := p.(*ast.SequentialProc); ok {
		return p
	}
	return &ast.SequentialProc{
		Procs: []ast.Proc{p},
	}
}

// liftFilter removes the filter at the head of the flowgraph AST, if
// one is present, and returns its ast.BooleanExpr and the modified
// flowgraph AST. If the flowgraph does not start with a filter, it
// returns nil and the unmodified flowgraph.
func liftFilter(p ast.Proc) (ast.BooleanExpr, ast.Proc) {
	if fp, ok := p.(*ast.FilterProc); ok {
		pass := &ast.PassProc{
			Node: ast.Node{"PassProc"},
		}
		return fp.Filter, pass
	}
	seq, ok := p.(*ast.SequentialProc)
	if ok && len(seq.Procs) > 0 {
		if fp, ok := seq.Procs[0].(*ast.FilterProc); ok {
			rest := &ast.SequentialProc{
				Node:  ast.Node{"SequentialProc"},
				Procs: seq.Procs[1:],
			}
			return fp.Filter, rest
		}
	}
	return nil, p
}

func ReplaceGroupByProcDurationWithKey(p ast.Proc) {
	switch p := p.(type) {
	case *ast.GroupByProc:
		if duration := p.Duration.Seconds; duration != 0 {
			durationKey := ast.ExpressionAssignment{
				Target: "ts",
				Expr: &ast.FunctionCall{
					Node:     ast.Node{"FunctionCall"},
					Function: "Time.trunc",
					Args: []ast.Expression{
						&ast.FieldRead{
							Node:  ast.Node{"FieldRead"},
							Field: "ts",
						},
						&ast.Literal{
							Node:  ast.Node{"Literal"},
							Type:  "int64",
							Value: strconv.Itoa(duration),
						}},
				},
			}
			p.Keys = append([]ast.ExpressionAssignment{durationKey}, p.Keys...)
		}
	case *ast.ParallelProc:
		for _, pp := range p.Procs {
			ReplaceGroupByProcDurationWithKey(pp)
		}
	case *ast.SequentialProc:
		for _, pp := range p.Procs {
			ReplaceGroupByProcDurationWithKey(pp)
		}
	}
}

// setGroupByProcInputSortDir examines p under the assumption that its input is
// sorted according to inputSortField and inputSortDir.  If p is an
// ast.GroupByProc and setGroupByProcInputSortDir can determine that its first
// grouping key is inputSortField or an order-preserving function of
// inputSortField, setGroupByProcInputSortDir sets ast.GroupByProc.InputSortDir
// to inputSortDir.  setGroupByProcInputSortDir returns true if it determines
// that p's output will remain sorted according to inputSortField and
// inputSortDir; otherwise, it returns false.
func setGroupByProcInputSortDir(p ast.Proc, inputSortField string, inputSortDir int) bool {
	switch p := p.(type) {
	case *ast.CutProc:
		// Return true if the output record contains inputSortField.
		for _, f := range p.Fields {
			if f.Source == inputSortField {
				return !p.Complement
			}
		}
		return p.Complement
	case *ast.GroupByProc:
		// Set p.InputSortDir and return true if the first grouping key
		// is inputSortField or an order-preserving function of it.
		if len(p.Keys) > 0 && p.Keys[0].Target == inputSortField {
			switch expr := p.Keys[0].Expr.(type) {
			case *ast.FieldRead:
				if expr.Field == inputSortField {
					p.InputSortDir = inputSortDir
					return true
				}
			case *ast.FunctionCall:
				switch expr.Function {
				case "Math.ceil", "Math.floor", "Math.round", "Time.trunc":
					if len(expr.Args) > 0 {
						arg0, ok := expr.Args[0].(*ast.FieldRead)
						if ok && arg0.Field == inputSortField {
							p.InputSortDir = inputSortDir
							return true
						}
					}
				}
			}
		}
		return false
	case *ast.PutProc:
		for _, c := range p.Clauses {
			if c.Target == inputSortField {
				return false
			}
		}
		return true
	case *ast.SequentialProc:
		for _, pp := range p.Procs {
			if !setGroupByProcInputSortDir(pp, inputSortField, inputSortDir) {
				return false
			}
		}
		return true
	case *ast.FilterProc, *ast.HeadProc, *ast.PassProc, *ast.UniqProc, *ast.TailProc, *ast.FuseProc:
		return true
	default:
		return false
	}
}

// expressionFields returns a slice with all fields referenced
// in an expression. Fields will be repeated if they appear
// repeatedly.
func expressionFields(e ast.Expression) []string {
	switch e := e.(type) {
	case *ast.UnaryExpression:
		return expressionFields(e.Operand)
	case *ast.BinaryExpression:
		return append(expressionFields(e.LHS), expressionFields(e.RHS)...)
	case *ast.ConditionalExpression:
		fields := expressionFields(e.Condition)
		fields = append(fields, expressionFields(e.Then)...)
		fields = append(fields, expressionFields(e.Else)...)
		return fields
	case *ast.FunctionCall:
		fields := []string{}
		for _, arg := range e.Args {
			fields = append(fields, expressionFields(arg)...)
		}
		return fields
	case *ast.CastExpression:
		return expressionFields(e.Expr)
	case *ast.Literal:
		return []string{}
	case *ast.FieldRead:
		return []string{e.Field}
	case *ast.FieldCall:
		return expressionFields(e.Field.(ast.Expression))
	default:
		panic("expression type not handled")
	}
}

// booleanExpressionFields returns a slice with all fields referenced
// in a boolean expression. Fields will be repeated if they appear
// repeatedly.  If all fields are referenced, nil is returned.
func booleanExpressionFields(e ast.BooleanExpr) []string {
	switch e := e.(type) {
	case *ast.Search:
		return nil
	case *ast.LogicalAnd:
		l := booleanExpressionFields(e.Left)
		r := booleanExpressionFields(e.Right)
		if l == nil || r == nil {
			return nil
		}
		return append(l, r...)
	case *ast.LogicalOr:
		l := booleanExpressionFields(e.Left)
		r := booleanExpressionFields(e.Right)
		if l == nil || r == nil {
			return nil
		}
		return append(l, r...)
	case *ast.LogicalNot:
		return booleanExpressionFields(e.Expr)
	case *ast.MatchAll:
		return []string{}
	case *ast.CompareAny:
		return nil
	case *ast.CompareField:
		return expressionFields(e.Field.(ast.Expression))
	default:
		panic("boolean expression type not handled")
	}
}

// computeColumns walks a flowgraph and computes a subset of columns
// that can be read by the source without modifying the output. For
// example, for the flowgraph "* | cut x", only the column "x" needs
// to be read by the source. On the other hand, for the flowgraph "* >
// 1", all columns need to be read.
//
// The return value is a map where the keys are string representations
// of the columns to be read at the source. If the return value is a
// nil map, all columns must be read.
func computeColumns(p ast.Proc) map[string]struct{} {
	cols, _ := computeColumnsR(p, map[string]struct{}{})
	return cols
}

// computeColumnsR is the recursive func used by computeColumns to
// compute a column set that can be read at the source. It walks a
// flowgraph, from the source, until it hits a "boundary proc". A
// "boundary proc" is one for which we can identify a set of input columns
// that fully determine its output. For example, 'cut x' is boundary
// proc (with set {x}); 'filter *>1' is a boundary proc (with set "all
// fields"); and 'head' is not a boundary proc.
// The first return value is a map representing the column set; the
// second is bool indicating that a boundary proc has been reached.
//
// Note that this function does not calculate the smallest column set
// for all possible flowgraphs: (1) It does not walk into parallel
// procs. (2) It does not track field renames: 'rename foo=y | count()
// by x' gets the column set {x, y} which is greater than the minimal
// column set {x}. (However 'rename x=y | count() by x' also gets {x,
// y}, which is minimal).
func computeColumnsR(p ast.Proc, colset map[string]struct{}) (map[string]struct{}, bool) {
	switch p := p.(type) {
	case *ast.CutProc:
		if p.Complement {
			return colset, false
		}
		for _, f := range p.Fields {
			colset[f.Source] = struct{}{}
		}
		return colset, true
	case *ast.GroupByProc:
		for _, r := range p.Reducers {
			if r.Field != nil {
				colset[expr.FieldExprToString(r.Field)] = struct{}{}
			}
		}
		for _, key := range p.Keys {
			for _, field := range expressionFields(key.Expr) {
				colset[field] = struct{}{}
			}
		}
		return colset, true
	case *ast.SequentialProc:
		for _, pp := range p.Procs {
			var done bool
			colset, done = computeColumnsR(pp, colset)
			if done {
				return colset, true
			}
		}
		// We got to end without seeing a boundary proc, return "all cols"
		return nil, true
	case *ast.ParallelProc:
		// (These could be further analysed to determine the
		// colsets on each branch, and then merge them at the
		// split point.)
		return nil, true
	case *ast.UniqProc, *ast.FuseProc:
		return nil, true
	case *ast.HeadProc, *ast.TailProc, *ast.PassProc:
		return colset, false
	case *ast.FilterProc:
		fields := booleanExpressionFields(p.Filter)
		if fields == nil {
			return nil, true
		}
		for _, field := range fields {
			colset[field] = struct{}{}
		}
		return colset, false
	case *ast.PutProc:
		for _, c := range p.Clauses {
			for _, field := range expressionFields(c.Expr) {
				colset[field] = struct{}{}
			}
		}
		return colset, false
	case *ast.RenameProc:
		for _, f := range p.Fields {
			colset[f.Source] = struct{}{}
		}
		return colset, false
	case *ast.SortProc:
		if len(p.Fields) == 0 {
			// we don't know which sort field will
			// be used.
			return nil, true
		}
		for _, f := range p.Fields {
			colset[expr.FieldExprToString(f)] = struct{}{}
		}
		return colset, false
	default:
		panic("proc type not handled")
	}
}

func copyProcs(ps []ast.Proc) []ast.Proc {
	var copies []ast.Proc
	for _, p := range ps {
		b, err := json.Marshal(p)
		if err != nil {
			panic(err)
		}
		proc, err := ast.UnpackJSON(nil, b)
		if err != nil {
			panic(err)
		}
		copies = append(copies, proc)
	}
	return copies
}

func buildSplitFlowgraph(branch, tail []ast.Proc, mergeField string, reverse bool, N int) *ast.SequentialProc {
	if len(tail) == 0 && mergeField != "" {
		// Insert a pass tail in order to force a merge of the
		// parallel branches when compiling. (Trailing parallel branches are wired to
		// a mux output).
		tail = []ast.Proc{&ast.PassProc{Node: ast.Node{"PassProc"}}}
	}
	pp := &ast.ParallelProc{
		Node:              ast.Node{"ParallelProc"},
		Procs:             []ast.Proc{},
		MergeOrderField:   mergeField,
		MergeOrderReverse: reverse,
	}
	for i := 0; i < N; i++ {
		pp.Procs = append(pp.Procs, &ast.SequentialProc{
			Node:  ast.Node{"SequentialProc"},
			Procs: copyProcs(branch),
		})
	}
	return &ast.SequentialProc{
		Node:  ast.Node{"SequentialProc"},
		Procs: append([]ast.Proc{pp}, tail...),
	}
}

func decomposable(rs []ast.Reducer) bool {
	for _, r := range rs {
		cr, err := rcompile.Compile(r)
		if err != nil {
			return false
		}
		if _, ok := cr.Instantiate().(reducer.Decomposable); !ok {
			return false
		}
	}
	return true
}

// parallelizeFlowgraph takes a sequential proc AST and tries to
// parallelize it by splitting as much as possible of the sequence
// into N parallel branches. The boolean return argument indicates
// whether the flowgraph could be parallelized.
func parallelizeFlowgraph(seq *ast.SequentialProc, N int, inputSortField string, inputSortReversed bool) (*ast.SequentialProc, bool) {
	orderSensitiveTail := true
	for i := range seq.Procs {
		switch seq.Procs[i].(type) {
		case *ast.SortProc, *ast.GroupByProc:
			orderSensitiveTail = false
			break
		default:
			continue
		}
	}
	for i := range seq.Procs {
		switch p := seq.Procs[i].(type) {
		case *ast.FilterProc, *ast.PassProc:
			// Stateless procs: continue until we reach one of the procs below at
			// which point we'll either split the flowgraph or see we can't and return it as-is.
			continue
		case *ast.CutProc:
			if inputSortField == "" || !orderSensitiveTail {
				continue
			}
			if p.Complement {
				for _, f := range p.Fields {
					if f.Source == inputSortField {
						return buildSplitFlowgraph(seq.Procs[0:i], seq.Procs[i:], inputSortField, inputSortReversed, N), true
					}
				}
				continue
			}
			var found bool
			for _, f := range p.Fields {
				if f.Source != inputSortField && f.Target == inputSortField {
					return buildSplitFlowgraph(seq.Procs[0:i], seq.Procs[i:], inputSortField, inputSortReversed, N), true
				}
				if f.Source == inputSortField && f.Target == "" {
					found = true
				}
			}
			if !found {
				return buildSplitFlowgraph(seq.Procs[0:i], seq.Procs[i:], inputSortField, inputSortReversed, N), true
			}
		case *ast.PutProc:
			if inputSortField == "" || !orderSensitiveTail {
				continue
			}
			for _, c := range p.Clauses {
				if c.Target == inputSortField {
					return buildSplitFlowgraph(seq.Procs[0:i], seq.Procs[i:], inputSortField, inputSortReversed, N), true
				}
			}
			continue
		case *ast.RenameProc:
			if inputSortField == "" || !orderSensitiveTail {
				continue
			}
			for _, f := range p.Fields {
				if f.Target == inputSortField {
					return buildSplitFlowgraph(seq.Procs[0:i], seq.Procs[i:], inputSortField, inputSortReversed, N), true
				}
			}
		case *ast.GroupByProc:
			if !decomposable(p.Reducers) {
				return buildSplitFlowgraph(seq.Procs[0:i], seq.Procs[i:], inputSortField, inputSortReversed, N), true
			}
			// We have a decomposable groupby and can split the flowgraph into branches that run up to and including a groupby,
			// followed by a post-merge groupby that composes the results.
			var mergeField string
			if p.Duration.Seconds != 0 {
				// Group by time requires a time-ordered merge, irrespective of any upstream ordering.
				mergeField = "ts"
			}
			branch := copyProcs(seq.Procs[0 : i+1])
			branch[len(branch)-1].(*ast.GroupByProc).EmitPart = true

			composerGroupBy := copyProcs([]ast.Proc{p})[0].(*ast.GroupByProc)
			composerGroupBy.ConsumePart = true

			return buildSplitFlowgraph(branch, append([]ast.Proc{composerGroupBy}, seq.Procs[i+1:]...), mergeField, false, N), true
		case *ast.SortProc:
			dir := map[int]bool{-1: true, 1: false}[p.SortDir]
			if len(p.Fields) == 1 {
				// Single sort field: we can sort in each parallel branch, and then do an ordered merge.
				mergeField := expr.FieldExprToString(p.Fields[0])
				return buildSplitFlowgraph(seq.Procs[0:i+1], seq.Procs[i+1:], mergeField, dir, N), true
			} else {
				// Unknown or multiple sort fields: we sort after the merge point, which can be unordered.
				return buildSplitFlowgraph(seq.Procs[0:i], seq.Procs[i:], "", dir, N), true
			}
		case *ast.ParallelProc:
			return seq, false
		case *ast.HeadProc, *ast.TailProc:
			if inputSortField == "" {
				// Unknown order: we can't parallelize because we can't maintain this unknown order at the merge point.
				return seq, false
			}
			// put one head/tail on each parallel branch and one after the merge.
			return buildSplitFlowgraph(seq.Procs[0:i+1], seq.Procs[i:], inputSortField, inputSortReversed, N), true
		case *ast.UniqProc, *ast.FuseProc:
			if inputSortField == "" {
				// Unknown order: we can't parallelize because we can't maintain this unknown order at the merge point.
				return seq, false
			}
			// Split all the upstream procs into parallel branches, then merge and continue with this and any remaining procs.
			return buildSplitFlowgraph(seq.Procs[0:i], seq.Procs[i:], inputSortField, inputSortReversed, N), true
		case *ast.SequentialProc:
			return seq, false
		default:
			panic("proc type not handled")
		}
	}
	// If we're here, we reached the end of the flowgraph without
	// coming across a merge-forcing proc. If inputs are sorted,
	// we can parallelize the entire chain and do an ordered
	// merge. Otherwise, no parallelization.
	if inputSortField == "" {
		return seq, false
	}
	return buildSplitFlowgraph(seq.Procs, nil, inputSortField, inputSortReversed, N), true
}

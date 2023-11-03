package optimizer

import (
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/optimizer/demand"
)

// Returns a map from op to the demand on the output of that op.
func InferDemandSeqOut(seq dag.Seq) map[dag.Op]demand.Demand {
	demands := make(map[dag.Op]demand.Demand)
	inferDemandSeqOutWith(demands, demand.All{}, seq)
	for _, d := range demands {
		if !demand.IsValid(d) {
			panic("Invalid demand")
		}
	}
	return demands
}

func inferDemandSeqOutWith(demands map[dag.Op]demand.Demand, demandSeqOut demand.Demand, seq dag.Seq) {
	demandOpOut := demandSeqOut
	for i := len(seq) - 1; i >= 0; i-- {
		op := seq[i]
		if _, ok := demands[op]; ok {
			panic("Duplicate op value")
		}
		demands[op] = demandOpOut

		// Infer the demand that `op` places on it's input.
		var demandOpIn demand.Demand
		switch op := op.(type) {
		case *dag.FileScan:
			demandOpIn = demand.None()
		case *dag.Filter:
			demandOpIn = demand.Union(
				// Everything that downstream operations need.
				demandOpOut,
				// Everything that affects the outcome of this filter.
				inferDemandExprIn(demand.All{}, op.Expr),
			)
		case *dag.Yield:
			demandOpIn = demand.None()
			for _, expr := range op.Exprs {
				demandOpIn = demand.Union(demandOpIn, inferDemandExprIn(demandOpOut, expr))
			}
		default:
			// Conservatively assume that `op` uses it's entire input, regardless of output demand.
			_ = op
			demandOpIn = demand.All{}
		}
		demandOpOut = demandOpIn
	}
}

func inferDemandExprIn(demandExprOut demand.Demand, expr dag.Expr) demand.Demand {
	if demand.IsNone(demandExprOut) {
		return demandExprOut
	}
	switch expr := expr.(type) {
	case *dag.BinaryExpr:
		// Since we don't know how the expr.Op will transform the inputs, we have to assume demand.All.
		return demand.Union(
			inferDemandExprIn(demand.All{}, expr.LHS),
			inferDemandExprIn(demand.All{}, expr.RHS),
		)
	case *dag.Dot:
		return demand.Key(expr.RHS, inferDemandExprIn(demandExprOut, expr.LHS))
	case *dag.Literal:
		return demand.None()
	case *dag.MapExpr:
		demandExprIn := demand.None()
		for _, entry := range expr.Entries {
			demandExprIn = demand.Union(demandExprIn, inferDemandExprIn(demand.All{}, entry.Key))
			demandExprIn = demand.Union(demandExprIn, inferDemandExprIn(demand.All{}, entry.Value))
		}
		return demandExprIn
	case *dag.RecordExpr:
		demandExprIn := demand.None()
		for _, elem := range expr.Elems {
			switch elem := elem.(type) {
			case *dag.Field:
				demandValueOut := demand.GetKey(demandExprOut, elem.Name)
				if !demand.IsNone(demandValueOut) {
					demandExprIn = demand.Union(demandExprIn, inferDemandExprIn(demandValueOut, elem.Value))
				}
			case *dag.Spread:
				demandExprIn = demand.Union(demandExprIn, inferDemandExprIn(demand.All{}, elem.Expr))
			}
		}
		return demandExprIn
	case *dag.This:
		demandExprIn := demandExprOut
		for i := len(expr.Path) - 1; i >= 0; i-- {
			demandExprIn = demand.Key(expr.Path[i], demandExprIn)
		}
		return demandExprIn
	default:
		// Conservatively assume that `expr` uses it's entire input, regardless of output demand.
		return demand.All{}
	}
}

// --- The functions below are used for testing demand inference and can be removed once demand is used to prune inputs. ---

func insertDemandTests(seq dag.Seq) dag.Seq {
	demands := InferDemandSeqOut(seq)
	return walk(seq, true, func(seq dag.Seq) dag.Seq {
		ops := make([]dag.Op, 0, 2*len(seq))
		for _, op := range seq {
			ops = append(ops, op)
			if demand, ok := demands[op]; ok {
				// We can't insert anything after a Fork.
				if _, ok := op.(*dag.Fork); !ok {
					testOp := dag.Yield{
						Kind:  "Yield",
						Exprs: []dag.Expr{yieldExprFromDemand(demand, []string{})},
					}
					ops = append(ops, &testOp)
				}
			}
		}
		return ops
	})
}

func yieldExprFromDemand(demandOut demand.Demand, path []string) dag.Expr {
	switch demandOut := demandOut.(type) {
	case demand.All:
		return &dag.This{Kind: "This", Path: path}
	case demand.Keys:
		elems := make([]dag.RecordElem, 0, len(demandOut))
		for key, demandKeyOut := range demandOut {
			keyPath := append(append(make([]string, 0, len(path)+1), path...), key)
			elems = append(elems, &dag.Field{
				Kind:  "Field",
				Name:  key,
				Value: yieldExprFromDemand(demandKeyOut, keyPath),
			})
		}
		return &dag.RecordExpr{
			Kind:  "RecordExpr",
			Elems: elems,
		}
	default:
		panic("Unreachable")
	}
}

package optimizer

import (
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/compiler/optimizer/demand"
)

func insertDemand(seq dag.Seq) dag.Seq {
	demands := InferDemandSeqOut(seq)
	return walk(seq, true, func(seq dag.Seq) dag.Seq {
		for _, op := range seq {
			if s, ok := op.(*dag.SeqScan); ok {
				s.Fields = demand.Fields(demands[op])
			}
		}
		return seq
	})
}

// Returns a map from op to the demand on the output of that op.
func InferDemandSeqOut(seq dag.Seq) map[dag.Op]demand.Demand {
	demands := make(map[dag.Op]demand.Demand)
	inferDemandSeqOutWith(demands, demand.All(), seq)
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
		case *dag.Filter:
			demandOpIn = demand.Union(
				// Everything that downstream operations need.
				demandOpOut,
				// Everything that affects the outcome of this filter.
				inferDemandExprIn(demand.All(), op.Expr),
			)
		case *dag.Summarize:
			demandOpIn = demand.None()
			// TODO If LHS not in demandOut, we can ignore RHS
			for _, assignment := range op.Keys {
				demandOpIn = demand.Union(demandOpIn, inferDemandExprIn(demand.All(), assignment.RHS))
			}
			for _, assignment := range op.Aggs {
				demandOpIn = demand.Union(demandOpIn, inferDemandExprIn(demand.All(), assignment.RHS))
			}
		case *dag.Yield:
			demandOpIn = demand.None()
			for _, expr := range op.Exprs {
				demandOpIn = demand.Union(demandOpIn, inferDemandExprIn(demandOpOut, expr))
			}
		default:
			// Conservatively assume that `op` uses it's entire input, regardless of output demand.
			demandOpIn = demand.All()
		}
		demandOpOut = demandOpIn
	}
}

func inferDemandExprIn(demandOut demand.Demand, expr dag.Expr) demand.Demand {
	if demand.IsNone(demandOut) {
		return demand.None()
	}
	if expr == nil {
		return demand.None()
	}
	var demandIn demand.Demand
	switch expr := expr.(type) {
	case *dag.Agg:
		// Since we don't know how the expr.Name will transform the inputs, we have to assume demand.All.
		return demand.Union(
			inferDemandExprIn(demand.All(), expr.Expr),
			inferDemandExprIn(demand.All(), expr.Where),
		)
	case *dag.BinaryExpr:
		// Since we don't know how the expr.Op will transform the inputs, we have to assume demand.All.
		demandIn = demand.Union(
			inferDemandExprIn(demand.All(), expr.LHS),
			inferDemandExprIn(demand.All(), expr.RHS),
		)
	case *dag.Dot:
		demandIn = demand.Key(expr.RHS, inferDemandExprIn(demandOut, expr.LHS))
	case *dag.Literal:
		demandIn = demand.None()
	case *dag.MapExpr:
		demandIn = demand.None()
		for _, entry := range expr.Entries {
			demandIn = demand.Union(demandIn, inferDemandExprIn(demand.All(), entry.Key))
			demandIn = demand.Union(demandIn, inferDemandExprIn(demand.All(), entry.Value))
		}
	case *dag.RecordExpr:
		demandIn = demand.None()
		for _, elem := range expr.Elems {
			switch elem := elem.(type) {
			case *dag.Field:
				demandValueOut := demand.GetKey(demandOut, elem.Name)
				if !demand.IsNone(demandValueOut) {
					demandIn = demand.Union(demandIn, inferDemandExprIn(demandValueOut, elem.Value))
				}
			case *dag.Spread:
				demandIn = demand.Union(demandIn, inferDemandExprIn(demand.All(), elem.Expr))
			}
		}
	case *dag.This:
		demandIn = demandOut
		for i := len(expr.Path) - 1; i >= 0; i-- {
			demandIn = demand.Key(expr.Path[i], demandIn)
		}
	default:
		// Conservatively assume that `expr` uses it's entire input, regardless of output demand.
		demandIn = demand.All()
	}
	return demandIn
}

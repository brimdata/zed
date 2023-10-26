package optimizer

import (
	"fmt"
	"github.com/brimdata/zed/compiler/ast/dag"
)

type Demand interface {
	isDemand()
}

// TODO Think about normal form for Demand.

// A nil Demand means 'demand nothing'.
type DemandAll struct{}
type DemandKey struct {
	Key   string
	Value Demand // Not nil
}
type DemandUnion [2]Demand // Not nil or DemandAll

func (demand DemandAll) isDemand()   {}
func (demand DemandKey) isDemand()   {}
func (demand DemandUnion) isDemand() {}

func demandKey(key string, value Demand) Demand {
	if value == nil {
		return nil
	}
	return DemandKey{
		Key:   key,
		Value: value,
	}
}

func demandUnion(a Demand, b Demand) Demand {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	if _, ok := a.(DemandAll); ok {
		return a
	}
	if _, ok := b.(DemandAll); ok {
		return b
	}
	return DemandUnion([2]Demand{a, b})
}

func demandForSeq(seq dag.Seq) map[*dag.Op]Demand {
	demands := make(map[*dag.Op]Demand)
	demandForSeqInto(demands, DemandAll{}, seq)

	walk(seq, true, func(seq dag.Seq) dag.Seq {
		for i := range seq {
			fmt.Println(seq[i], " ", demands[&seq[i]])
		}
		return seq
	})

	return demands
}

func demandForSeqInto(demands map[*dag.Op]Demand, demandOnSeq Demand, seq dag.Seq) {
	var demand = demandOnSeq
	for i := len(seq) - 1; i >= 0; i-- {
		op_ptr := &seq[i]
		demands[op_ptr] = demand

		// Infer the demand that `op` places on it's input.
		switch op := (*op_ptr).(type) {
		case *dag.FileScan:
			demand = nil
		case *dag.Filter:
			demand = demandUnion(
				// Everything that downstream operations need.
				demand,
				// Everything that affects the outcome of this filter.
				demandForExpr(DemandAll{}, op.Expr),
			)
		case *dag.Yield:
			yieldDemand := demand
			demand = nil
			for _, expr := range op.Exprs {
				demand = demandUnion(demand, demandForExpr(yieldDemand, expr))
			}
		default:
			// Conservatively assume that `op` uses it's entire input, regardless of output demand.
			_ = op
			demand = DemandAll{}
		}
	}
}

func demandForExpr(demandOnExpr Demand, expr dag.Expr) Demand {
	if demandOnExpr == nil {
		return nil
	}
	switch expr := expr.(type) {
	case *dag.BinaryExpr:
		// Since we don't know how the expr.Op will transform the inputs, we have to assume DemandAll.
		return demandUnion(
			demandForExpr(DemandAll{}, expr.LHS),
			demandForExpr(DemandAll{}, expr.RHS),
		)
	case *dag.Dot:
		return demandKey(expr.RHS, demandForExpr(demandOnExpr, expr.LHS))
	case *dag.Literal:
		return nil
	case *dag.MapExpr:
		var demand Demand = nil
		for _, entry := range expr.Entries {
			demand = demandUnion(demand, demandForExpr(DemandAll{}, entry.Key))
			demand = demandUnion(demand, demandForExpr(DemandAll{}, entry.Value))
		}
		return demand
	case *dag.RecordExpr:
		var demand Demand = nil
		for _, elem := range expr.Elems {
			switch elem := elem.(type) {
			case *dag.Field:
				if d := demandForKey(demandOnExpr, elem.Name); d != nil {
					demand = demandUnion(demand, demandForExpr(d, elem.Value))
				}
			case *dag.Spread:
				demand = demandUnion(demand, demandForExpr(demandOnExpr, elem.Expr))
			}
		}
		return demand
	case *dag.This:
		var demand Demand = demandOnExpr
		for i := len(expr.Path) - 1; i >= 0; i-- {
			demand = demandKey(expr.Path[i], demand)
		}
		return demand
	default:
		// Conservatively assume that `expr` uses it's entire input, regardless of output demand.
		return DemandAll{}
	}
}

func demandForKey(demand Demand, key string) Demand {
	switch demand := demand.(type) {
	case nil:
		return nil
	case DemandAll:
		return demand
	case DemandKey:
		if key == demand.Key {
			return demand.Value
		} else {
			return nil
		}
	case DemandUnion:
		return demandUnion(
			demandForKey(demand[0], key),
			demandForKey(demand[1], key),
		)
	default:
		panic("Unreachable")
	}
}

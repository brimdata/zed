package optimizer

import (
	//"encoding/json"
	//"fmt"
	"github.com/brimdata/zed/compiler/ast/dag"
)

type DemandAll struct{}
type DemandKeys map[string]Demand // No empty values.

type Demand interface {
	isDemand()
}

func (demand DemandAll) isDemand()  {}
func (demand DemandKeys) isDemand() {}

func demandNone() Demand {
	return DemandKeys(make(map[string]Demand, 0))
}

func demandIsValid(demand Demand) bool {
	switch demand := demand.(type) {
	case DemandAll:
		return true
	case DemandKeys:
		for _, v := range demand {
			if !demandIsValid(v) || demandIsEmpty(v) {
				return false
			}
		}
		return true
	default:
		panic("Unreachable")
	}
}

func demandIsEmpty(demand Demand) bool {
	switch demand := demand.(type) {
	case DemandAll:
		return false
	case DemandKeys:
		return len(demand) == 0
	default:
		panic("Unreachable")
	}
}

func demandKey(key string, value Demand) Demand {
	if demandIsEmpty(value) {
		return value
	}
	demand := DemandKeys(make(map[string]Demand, 1))
	demand[key] = value
	return demand
}

func demandUnion(a Demand, b Demand) Demand {
	if _, ok := a.(DemandAll); ok {
		return a
	}
	if _, ok := b.(DemandAll); ok {
		return b
	}

	{
		a := a.(DemandKeys)
		b := b.(DemandKeys)

		demand := DemandKeys(make(map[string]Demand, len(a)+len(b)))
		for k, v := range a {
			demand[k] = v
		}
		for k, v := range b {
			if v2, ok := a[k]; ok {
				demand[k] = demandUnion(v, v2)
			} else {
				demand[k] = v
			}
		}
		return demand
	}
}

func demandForSeq(seq dag.Seq) map[dag.Op]Demand {
	demands := make(map[dag.Op]Demand)
	demandForSeqInto(demands, DemandAll{}, seq)

	//walk(seq, true, func(seq dag.Seq) dag.Seq {
	//    for i := range seq {
	//        fmt.Println(seq[i], " ", demands[seq[i]])
	//    }
	//    return seq
	//})

	for _, demand := range demands {
		if !demandIsValid(demand) {
			panic("Invalid demand")
		}
	}
	return demands
}

func demandForSeqInto(demands map[dag.Op]Demand, demandOnSeq Demand, seq dag.Seq) {
	var demand = demandOnSeq
	for i := len(seq) - 1; i >= 0; i-- {
		op := seq[i]
		if _, ok := demands[op]; ok {
			panic("Duplicate op value")
		}
		demands[op] = demand

		// Infer the demand that `op` places on it's input.
		switch op := op.(type) {
		case *dag.FileScan:
			demand = demandNone()
		case *dag.Filter:
			demand = demandUnion(
				// Everything that downstream operations need.
				demand,
				// Everything that affects the outcome of this filter.
				demandForExpr(DemandAll{}, op.Expr),
			)
		case *dag.Yield:
			yieldDemand := demand
			demand = demandNone()
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
	if demandIsEmpty(demandOnExpr) {
		return demandOnExpr
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
		return demandNone()
	case *dag.MapExpr:
		var demand Demand = demandNone()
		for _, entry := range expr.Entries {
			demand = demandUnion(demand, demandForExpr(DemandAll{}, entry.Key))
			demand = demandUnion(demand, demandForExpr(DemandAll{}, entry.Value))
		}
		return demand
	case *dag.RecordExpr:
		var demand Demand = demandNone()
		for _, elem := range expr.Elems {
			switch elem := elem.(type) {
			case *dag.Field:
				if d := demandForKey(demandOnExpr, elem.Name); !demandIsEmpty(d) {
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
	case DemandAll:
		return demand
	case DemandKeys:
		if value, ok := demand[key]; ok {
			return value
		} else {
			return demandNone()
		}
	default:
		panic("Unreachable")
	}
}

// --- The functions below are used for testing demand inference and can be removed once demand is used to prune inputs. ---

func insertDemandTests(seq dag.Seq) dag.Seq {
	demands := demandForSeq(seq)
	result := walk(seq, true, func(seq dag.Seq) dag.Seq {
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

	//b, _ := json.MarshalIndent(result, "", "    ")
	//fmt.Println(string(b))

	return result
}

func yieldExprFromDemand(demand Demand, path []string) dag.Expr {
	switch demand := demand.(type) {
	case DemandAll:
		return &dag.This{Kind: "This", Path: path}
	case DemandKeys:
		var elems = make([]dag.RecordElem, 0, len(demand))
		for key, keyDemand := range demand {
			keyPath := append(append(make([]string, 0, len(path)+1), path...), key)
			elems = append(elems, &dag.Field{
				Kind:  "Field",
				Name:  key,
				Value: yieldExprFromDemand(keyDemand, keyPath),
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

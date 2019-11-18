package ast

func (p *SequentialProc) Copy() Proc {
	var procs []Proc
	for _, proc := range p.Procs {
		procs = append(procs, proc.Copy())
	}
	return &SequentialProc{
		Node:  Node{p.Op},
		Procs: procs,
	}
}

func (p *ParallelProc) Copy() Proc {
	var procs []Proc
	for _, proc := range p.Procs {
		procs = append(procs, proc.Copy())
	}
	return &ParallelProc{Procs: procs}
}

func (p *CutProc) Copy() Proc {
	fields := make([]string, len(p.Fields))
	copy(fields, p.Fields)
	return &CutProc{
		Node:   Node{p.Op},
		Fields: fields,
	}
}

func (p *ReducerProc) Copy() Proc {
	reducers := make([]Reducer, len(p.Reducers))
	copy(reducers, p.Reducers)
	return &ReducerProc{
		Node:     Node{p.Op},
		Reducers: reducers,
	}
}

func (p *GroupByProc) Copy() Proc {
	reducers := make([]Reducer, len(p.Reducers))
	copy(reducers, p.Reducers)
	keys := make([]string, len(p.Keys))
	copy(keys, p.Keys)
	return &GroupByProc{
		Node:     Node{p.Op},
		Duration: p.Duration,
		Keys:     keys,
		Reducers: reducers,
	}
}

func (p *SortProc) Copy() Proc {
	fields := make([]string, len(p.Fields))
	copy(fields, p.Fields)
	return &SortProc{
		Node:    Node{p.Op},
		Fields:  fields,
		SortDir: p.SortDir,
	}
}

func (t *TopProc) Copy() Proc {
	return &TopProc{
		Node:   Node{t.Op},
		Limit:  t.Limit,
		Fields: append([]string{}, t.Fields...),
	}
}

func (p *HeadProc) Copy() Proc {
	copy := *p
	return &copy
}

func (p *TailProc) Copy() Proc {
	copy := *p
	return &copy
}

func (p *UniqProc) Copy() Proc {
	copy := *p
	return &copy
}

func (p *PassProc) Copy() Proc {
	copy := *p
	return &copy
}

func (p *FilterProc) Copy() Proc {
	return &FilterProc{
		Node:   Node{p.Op},
		Filter: p.Filter.Copy(),
	}
}

func (p *LogicalAnd) Copy() BooleanExpr {
	return &LogicalAnd{
		Node:  Node{p.Op},
		Left:  p.Left.Copy(),
		Right: p.Right.Copy(),
	}
}

func (p *LogicalOr) Copy() BooleanExpr {
	return &LogicalOr{
		Node:  Node{p.Op},
		Left:  p.Left.Copy(),
		Right: p.Right.Copy(),
	}
}

func (p *LogicalNot) Copy() BooleanExpr {
	return &LogicalNot{
		Node: Node{p.Op},
		Expr: p.Expr.Copy(),
	}
}

func (p *BooleanLiteral) Copy() BooleanExpr {
	copy := *p
	return &copy
}

func (p *CompareAny) Copy() BooleanExpr {
	copy := *p
	return &copy
}

func (p *CompareField) Copy() BooleanExpr {
	return &CompareField{
		Node:       Node{p.Op},
		Comparator: p.Comparator,
		Field:      p.Field.Copy(),
		Value:      p.Value,
	}
}

func (p *SearchString) Copy() BooleanExpr {
	copy := *p
	return &copy
}

func (p *FieldRead) Copy() FieldExpr {
	copy := *p
	return &copy
}

func (p *FieldCall) Copy() FieldExpr {
	copy := *p
	return &copy
}

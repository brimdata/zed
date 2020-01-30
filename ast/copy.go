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
	fields := make([]FieldExpr, len(p.Fields))
	for i, f := range p.Fields {
		fields[i] = f.Copy()
	}
	return &CutProc{
		Node:   Node{p.Op},
		Fields: fields,
	}
}

func (p *ReducerProc) Copy() Proc {
	reducers := make([]Reducer, len(p.Reducers))
	for i, r := range p.Reducers {
		reducers[i] = r.Copy()
	}
	return &ReducerProc{
		Node:     Node{p.Op},
		Reducers: reducers,
	}
}

func (p *GroupByProc) Copy() Proc {
	reducers := make([]Reducer, len(p.Reducers))
	for i, r := range p.Reducers {
		reducers[i] = r.Copy()
	}
	keys := make([]FieldExpr, len(p.Keys))
	for i, k := range p.Keys {
		keys[i] = k.Copy()
	}
	return &GroupByProc{
		Node:     Node{p.Op},
		Duration: p.Duration,
		Keys:     keys,
		Reducers: reducers,
	}
}

func (p *SortProc) Copy() Proc {
	fields := make([]FieldExpr, len(p.Fields))
	for i, f := range p.Fields {
		fields[i] = f.Copy()
	}
	return &SortProc{
		Node:    Node{p.Op},
		Fields:  fields,
		SortDir: p.SortDir,
	}
}

func (t *TopProc) Copy() Proc {
	fields := make([]FieldExpr, len(t.Fields))
	for i, f := range t.Fields {
		fields[i] = f.Copy()
	}
	return &TopProc{
		Node:   Node{t.Op},
		Limit:  t.Limit,
		Fields: fields,
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

func (b *LogicalAnd) Copy() BooleanExpr {
	return &LogicalAnd{
		Node:  Node{b.Op},
		Left:  b.Left.Copy(),
		Right: b.Right.Copy(),
	}
}

func (b *LogicalOr) Copy() BooleanExpr {
	return &LogicalOr{
		Node:  Node{b.Op},
		Left:  b.Left.Copy(),
		Right: b.Right.Copy(),
	}
}

func (b *LogicalNot) Copy() BooleanExpr {
	return &LogicalNot{
		Node: Node{b.Op},
		Expr: b.Expr.Copy(),
	}
}

func (b *BooleanLiteral) Copy() BooleanExpr {
	copy := *b
	return &copy
}

func (b *CompareAny) Copy() BooleanExpr {
	copy := *b
	return &copy
}

func (b *CompareField) Copy() BooleanExpr {
	return &CompareField{
		Node:       Node{b.Op},
		Comparator: b.Comparator,
		Field:      b.Field.Copy(),
		Value:      b.Value,
	}
}

func (f *FieldRead) Copy() FieldExpr {
	copy := *f
	return &copy
}

func (f *FieldCall) Copy() FieldExpr {
	copy := *f
	return &copy
}

func (r Reducer) Copy() Reducer {
	copy := r
	if r.Field != nil {
		copy.Field = r.Field.Copy()
	}
	return copy
}

package kernel

import (
	"github.com/brimsec/zq/field"
	"github.com/brimsec/zq/zng"
)

type DSL interface {
	dagNode()
}

type Operator interface {
	operator()
}

type Program struct {
	//XXX consts should get subsitited but types is needed by the runtime
	// to put any type names into the runtime type context.
	// FunctionDefs can be inlined too and should be a parser thing,
	// unless there is some reason that the modularlity would be useful...
	Consts    []Const
	Types     []Type
	Functions []FunctionDef // XXX TBD
	Entry     Operator
}

type Const struct {
	Op    string    `json:"op"`
	Name  string    `json:"name"`
	Value zng.Value `json:"value"`
}

type Type struct {
	Op    string `json:"op"`
	Name  string `json:"name"`
	Value zng.Value
}

// XXX TBD user-defined functions
type FunctionDef struct {
	Name     string       `json:"name"`
	Formals  []Identifier `json:"formals"`
	Function Expr         `json:"function"`
}

// ----------------------------------------------------------------------------
// Operators

type (
	Agg struct {
		Op          string          `json:"op"`
		Keys        []Assignment    `json:"keys"`
		Aggs        []AggAssignment `json:"aggs"`
		Duration    zng.Value       `json:"duration"`
		Limit       int             `json:"limit"`
		InputDesc   bool            `json:"input_desc"`
		PartialsIn  bool            `json:"partials_in,omitempty"`
		PartialsOut bool            `json:"partials_out,omitempty"`
	}
	Cut struct {
		Op          string       `json:"op"`
		Assignments []Assignment `json:"assignments"`
	}
	Drop struct {
		Op     string `json:"op"`
		Fields []Expr `json:"fields"`
	}
	Filter struct {
		Op     string `json:"op"`
		Filter Expr   `json:"filter"`
	}
	Fuse struct {
		Op string `json:"op"`
	}
	Head struct {
		Op    string `json:"op"`
		Count int    `json:"count"`
	}
	Join struct {
		Op          string       `json:"op"`
		LeftKey     Expr         `json:"left_key"`
		RightKey    Expr         `json:"right_key"`
		Assignments []Assignment `json:"assignments"`
	}
	Merge struct {
		Field   field.Static `json:"field"`
		Reverse bool         `json:"reverse"`
	}
	Parallel struct {
		Op        string     `json:"op"`
		Operators []Operator `json:"operators"`
	}
	Pass struct {
		Op string `json:"op"`
	}
	Pick struct {
		Op          string       `json:"op"`
		Assignments []Assignment `json:"assignments"`
	}
	Put struct {
		Op          string       `json:"op"`
		Assignments []Assignment `json:"assignments"`
	}
	Rename struct {
		Op          string       `json:"op"`
		Assignments []Assignment `json:"assignments"`
	}
	Sequential struct {
		Op        string     `json:"op"`
		Operators []Operator `json:"operators"`
	}
	Sort struct {
		Op         string `json:"op"`
		Fields     []Expr `json:"fields"`
		Reverse    bool   `json:"reverse"`
		NullsFirst bool   `json:"nullsfirst"`
	}
	Tail struct {
		Op    string `json:"op"`
		Count int    `json:"count"`
	}
	Top struct {
		Op     string `json:"op"`
		Limit  int    `json:"limit"`
		Fields []Expr `json:"fields"`
		Flush  bool   `json:"flush"`
	}
	Uniq struct {
		Op    string `json:"op"`
		Cflag bool   `json:"cflag"`
	}
)

type Assignment struct {
	LHS Expr `json:"lhs"`
	RHS Expr `json:"rhs"`
}

type AggAssignment struct {
	LHS Expr    `json:"lhs"`
	RHS AggFunc `json:"rhs"`
}

type AggFunc struct {
	Name  string   `json:"operator"`
	Arg   Expr     `json:"expr"`
	Where BoolExpr `json:"where"`
}

func (*Agg) operator()        {}
func (*Cut) operator()        {}
func (*Drop) operator()       {}
func (*Filter) operator()     {}
func (*Fuse) operator()       {}
func (*Head) operator()       {}
func (*Join) operator()       {}
func (*Merge) operator()      {}
func (*Parallel) operator()   {}
func (*Pass) operator()       {}
func (*Pick) operator()       {}
func (*Put) operator()        {}
func (*Rename) operator()     {}
func (*Sequential) operator() {}
func (*Sort) operator()       {}
func (*Tail) operator()       {}
func (*Top) operator()        {}
func (*Uniq) operator()       {}

// ----------------------------------------------------------------------------
// Expressions

type Expr interface {
	expr()
}

type BoolExpr interface {
	boolean()
	expr()
}

type (
	BinaryExpr struct {
		Op       string `json:"op"`
		Operator string `json:"operator"`
		LHS      Expr   `json:"lhs"`
		RHS      Expr   `json:"rhs"`
	}
	CallExpr struct {
		Op   string `json:"op"`
		Name string `json:"function"`
		Args []Expr `json:"args"`
	}
	CastExpr struct {
		Op   string `json:"op"`
		Expr Expr   `json:"expr"`
		Type string `json:"type"`
	}
	CondExpr struct {
		Op        string `json:"op"`
		Condition Expr   `json:"condition"`
		Then      Expr   `json:"then"`
		Else      Expr   `json:"else"`
	}
	ConstExpr struct {
		Op    string    `json:"op"`
		Value zng.Value `json:"value"`
	}
	Dot struct {
		Op string `json:"op"`
	}
	EmptyExpr struct {
		Op string `json:"op"`
	}
	Identifier struct {
		Op   string `json:"op"`
		Name string `json:"name"`
	}
	//XXX break out regexp, etc
	SearchExpr struct {
		Op   string `json:"op"`
		Text string `json:"text"`
		//XXX zng.Value doesn't work yet... need marshaling
		// and unmarshal requires ZSON parsing.
		Value zng.Value `json:"value"`
	}
	RegexpExpr struct {
		Op      string `json:"op"`
		Pattern string `json:"pattern"`
	}
	SeqExpr struct {
		Op        string   `json:"op"`
		Name      string   `json:"name"`
		Selectors []Expr   `json:"selectors"`
		Methods   []Method `json:"methods"`
	}
	UnaryExpr struct {
		Op       string `json:"op"`
		Operator string `json:"operator"`
		Operand  Expr   `json:"operand"`
	}
)

type Method struct {
	Name string `json:"name"`
	Args []Expr `json:"args"`
}

func (c *ConstExpr) IsNet() bool {
	return zng.AliasedType(c.Value.Type) == zng.TypeNet
}

func (*BinaryExpr) expr() {}
func (*CallExpr) expr()   {}
func (*CastExpr) expr()   {}
func (*CondExpr) expr()   {}
func (*ConstExpr) expr()  {}
func (*Dot) expr()        {}
func (*EmptyExpr) expr()  {}
func (*Identifier) expr() {}
func (*SearchExpr) expr() {}
func (*SeqExpr) expr()    {}
func (*UnaryExpr) expr()  {}

func (*UnaryExpr) boolean()  {}
func (*BinaryExpr) boolean() {}
func (*ConstExpr) boolean()  {}

func DotExprToField(n Expr) (field.Static, bool) {
	switch n := n.(type) {
	case nil:
		return nil, true
	case *BinaryExpr:
		if n.Operator == "." || n.Operator == "[" {
			lhs, ok := DotExprToField(n.LHS)
			if !ok {
				return nil, false
			}
			rhs, ok := DotExprToField(n.RHS)
			if !ok {
				return nil, false
			}
			return append(lhs, rhs...), true
		}
	case *Identifier:
		return field.Static{n.Name}, true
	case *Dot, *EmptyExpr:
		return nil, true
	}
	return nil, false
}

//XXX shouldn't need this as semantic analyzer uses this and it will be
// operating on the AST...?  or will it?  => semantic analyzer should
// generate an AST then optimizer operates on the kernel DSL => correct
func FieldsOf(e Expr) []field.Static {
	switch e := e.(type) {
	default:
		f, _ := DotExprToField(e)
		if f == nil {
			return nil
		}
		return []field.Static{f}
	case *BinaryExpr:
		if e.Operator == "." || e.Operator == "[" {
			lhs, _ := DotExprToField(e.LHS)
			rhs, _ := DotExprToField(e.RHS)
			var fields []field.Static
			if lhs != nil {
				fields = append(fields, lhs)
			}
			if rhs != nil {
				fields = append(fields, rhs)
			}
			return fields
		}
		return append(FieldsOf(e.LHS), FieldsOf(e.RHS)...)
		//XXX
		//case *Assignment:
		//	return append(FieldsOf(e.LHS), FieldsOf(e.RHS)...)
	case *SeqExpr:
		var fields []field.Static
		for _, selector := range e.Selectors {
			fields = append(fields, FieldsOf(selector)...)
		}
		for _, m := range e.Methods {
			for _, e := range m.Args {
				fields = append(fields, FieldsOf(e)...)
			}
		}
		return fields
	case *CallExpr:
		var fields []field.Static
		for _, e := range e.Args {
			fields = append(fields, FieldsOf(e)...)
		}
		return fields
	}
}

func NewDotExpr(f field.Static) Expr {
	lhs := Expr(&Dot{Op: "Dot"})
	for _, name := range f {
		rhs := &Identifier{
			Op:   "Identifier",
			Name: name,
		}
		lhs = &BinaryExpr{
			Op:       "BinaryExpr",
			Operator: ".",
			LHS:      lhs,
			RHS:      rhs,
		}
	}
	return lhs
}

//XXX not sure we need this
func NewAggAssignment(name string, lval field.Static, arg field.Static) AggAssignment {
	aggFunc := AggFunc{Name: name}
	if arg != nil {
		aggFunc.Arg = NewDotExpr(arg)
	}
	if lval == nil {
		panic("semantic analyzer should have filled this in")
	}
	return AggAssignment{LHS: NewDotExpr(lval), RHS: aggFunc}
}

func FanIn(o Operator) int {
	if seq, ok := o.(*Sequential); ok {
		return FanIn(seq.Operators[0])
	}
	if o, ok := o.(*Parallel); ok {
		return len(o.Operators)
	}
	if _, ok := o.(*Join); ok {
		return 2
	}
	return 1
}

func (p *Program) FanIn() int {
	return FanIn(p.Entry)
}

func FilterToProc(e Expr) *Filter {
	return &Filter{
		Op:     "Filter",
		Filter: e,
	}
}

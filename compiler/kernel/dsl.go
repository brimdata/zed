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
	Sequential struct {
		Op        string     `json:"op"`
		Operators []Operator `json:"operators"`
	}
	Parallel struct {
		Op        string     `json:"op"`
		Operators []Operator `json:"operators"`
	}
	Merge struct {
		Field   field.Static `json:"field"`
		Reverse bool         `json:"reverse"`
	}
	Sort struct {
		Op         string `json:"op"`
		Fields     []Expr `json:"fields"`
		Reverse    bool   `json:"reverse"`
		NullsFirst bool   `json:"nullsfirst"`
	}
	Cut struct {
		Op     string       `json:"op"`
		Fields []Assignment `json:"fields"`
	}
	Pick struct {
		Op     string       `json:"op"`
		Fields []Assignment `json:"fields"`
	}
	Drop struct {
		Op     string `json:"op"`
		Fields []Expr `json:"fields"`
	}
	Head struct {
		Op    string `json:"op"`
		Count int    `json:"count"`
	}
	Tail struct {
		Op    string `json:"op"`
		Count int    `json:"count"`
	}
	Filter struct {
		Op     string `json:"op"`
		Filter Expr   `json:"filter"`
	}
	Pass struct {
		Op string `json:"op"`
	}
	Uniq struct {
		Op    string `json:"op"`
		Cflag bool   `json:"cflag"`
	}
	Agg struct {
		Op           string          `json:"op"`
		Duration     zng.Value       `json:"duration"`
		InputSortDir int             `json:"input_sort_dir,omitempty"`
		Limit        int             `json:"limit"`
		Keys         []Assignment    `json:"keys"`
		Aggs         []AggAssignment `json:"aggs"`
		PartialsIn   bool            `json:"partials_in,omitempty"`
		PartialsOut  bool            `json:"partials_out,omitempty"`
	}
	Top struct {
		Op     string `json:"op"`
		Limit  int    `json:"limit"`
		Fields []Expr `json:"fields"`
		Flush  bool   `json:"flush"`
	}
	Put struct {
		Op      string       `json:"op"`
		Clauses []Assignment `json:"clauses"`
	}
	Rename struct {
		Op     string       `json:"op"`
		Fields []Assignment `json:"fields"`
	}
	Fuse struct {
		Op string `json:"op"`
	}
	Join struct {
		Op       string       `json:"op"`
		LeftKey  Expr         `json:"left_key"`
		RightKey Expr         `json:"right_key"`
		Clauses  []Assignment `json:"clauses"`
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

//XXX alphabetize
func (*Sequential) operator() {}
func (*Parallel) operator()   {}
func (*Sort) operator()       {}
func (*Cut) operator()        {}
func (*Pick) operator()       {}
func (*Drop) operator()       {}
func (*Head) operator()       {}
func (*Tail) operator()       {}
func (*Pass) operator()       {}
func (*Filter) operator()     {}
func (*Uniq) operator()       {}
func (*Agg) operator()        {}
func (*Top) operator()        {}
func (*Put) operator()        {}
func (*Rename) operator()     {}
func (*Fuse) operator()       {}
func (*Join) operator()       {}
func (*Merge) operator()      {}

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
	Identifier struct {
		Op   string `json:"op"`
		Name string `json:"name"`
	}
	Dot struct {
		Op string `json:"op"`
	}
	EmptyExpr struct {
		Op string `json:"op"`
	}
	//XXX break out regexp, etc
	SearchExpr struct {
		Op   string `json:"op"`
		Text string `json:"text"`
		//XXX zng.Value doesn't work
		Value zng.Value `json:"value"`
	}
	UnaryExpr struct {
		Op       string `json:"op"`
		Operator string `json:"operator"`
		Operand  Expr   `json:"operand"`
	}
	BinaryExpr struct {
		Op       string `json:"op"`
		Operator string `json:"operator"`
		LHS      Expr   `json:"lhs"`
		RHS      Expr   `json:"rhs"`
	}
	//XXX need to change this to not overlap with SQL
	SelectExpr struct {
		Op        string `json:"op"`
		Selectors []Expr `json:"selectors"`
	}
	ConstExpr struct {
		Op    string    `json:"op"`
		Value zng.Value `json:"value"`
	}
	CondExpr struct {
		Op        string `json:"op"`
		Condition Expr   `json:"condition"`
		Then      Expr   `json:"then"`
		Else      Expr   `json:"else"`
	}
	CallExpr struct {
		Op   string `json:"op"`
		Name string `json:"function"`
		Args []Expr `json:"args"`
	}
	MethodExpr struct {
		Op   string `json:"op"`
		Name string `json:"function"`
		Args []Expr `json:"args"`
	}
	CastExpr struct {
		Op   string `json:"op"`
		Expr Expr   `json:"expr"`
		Type string `json:"type"`
	}
)

func (*UnaryExpr) expr()  {}
func (*BinaryExpr) expr() {}
func (*SelectExpr) expr() {}
func (*CondExpr) expr()   {}
func (*SearchExpr) expr() {}
func (*CallExpr) expr()   {}
func (*CastExpr) expr()   {}
func (*ConstExpr) expr()  {}
func (*Identifier) expr() {}
func (*Dot) expr()        {}
func (*EmptyExpr) expr()  {}
func (*Assignment) expr() {}

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
	case *Assignment:
		return append(FieldsOf(e.LHS), FieldsOf(e.RHS)...)
	case *SelectExpr:
		var fields []field.Static
		for _, selector := range e.Selectors {
			fields = append(fields, FieldsOf(selector)...)
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

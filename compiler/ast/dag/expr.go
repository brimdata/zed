package dag

type Expr interface {
	ExprDAG()
}

// Exprs

type (
	ArrayExpr struct {
		Kind  string       `json:"kind" unpack:""`
		Elems []VectorElem `json:"elems"`
	}
	Assignment struct {
		Kind string `json:"kind" unpack:""`
		LHS  Expr   `json:"lhs"`
		RHS  Expr   `json:"rhs"`
	}
	BinaryExpr struct {
		Kind string `json:"kind" unpack:""`
		Op   string `json:"op"`
		LHS  Expr   `json:"lhs"`
		RHS  Expr   `json:"rhs"`
	}
	Call struct {
		Kind string `json:"kind" unpack:""`
		Name string `json:"name"`
		Args []Expr `json:"args"`
	}
	Conditional struct {
		Kind string `json:"kind" unpack:""`
		Cond Expr   `json:"cond"`
		Then Expr   `json:"then"`
		Else Expr   `json:"else"`
	}
	Dot struct {
		Kind string `json:"kind" unpack:""`
		LHS  Expr   `json:"lhs"`
		RHS  string `json:"rhs"`
	}
	MapExpr struct {
		Kind    string  `json:"kind" unpack:""`
		Entries []Entry `json:"entries"`
	}
	This struct {
		Kind string   `json:"kind" unpack:""`
		Path []string `json:"path"`
	}
	RecordExpr struct {
		Kind  string       `json:"kind" unpack:""`
		Elems []RecordElem `json:"elems"`
	}
	RecordElem interface {
		recordAST()
	}
	Literal struct {
		Kind  string `json:"kind" unpack:""`
		Value string `json:"value"`
	}
	Var struct {
		Kind string `json:"kind" unpack:""`
		Name string `json:"name"`
		Slot int    `json:"slot"`
	}
	RegexpMatch struct {
		Kind    string `json:"kind" unpack:""`
		Pattern string `json:"pattern"`
		Expr    Expr   `json:"expr"`
	}
	RegexpSearch struct {
		Kind    string `json:"kind" unpack:""`
		Pattern string `json:"pattern"`
		Expr    Expr   `json:"expr"`
	}
	Search struct {
		Kind  string `json:"kind" unpack:""`
		Text  string `json:"text"`
		Value string `json:"value"`
		Expr  Expr   `json:"expr"`
	}
	SetExpr struct {
		Kind  string       `json:"kind" unpack:""`
		Elems []VectorElem `json:"elems"`
	}
	UnaryExpr struct {
		Kind    string `json:"kind" unpack:""`
		Op      string `json:"op"`
		Operand Expr   `json:"operand"`
	}
	OverExpr struct {
		Kind  string      `json:"kind" unpack:""`
		Defs  []Def       `json:"defs"`
		Exprs []Expr      `json:"exprs"`
		Scope *Sequential `json:"scope"`
	}
	VectorElem interface {
		vectorElem()
	}
)

func NewBinaryExpr(op string, lhs, rhs Expr) *BinaryExpr {
	return &BinaryExpr{
		Kind: "BinaryExpr",
		Op:   op,
		LHS:  lhs,
		RHS:  rhs,
	}
}

func (*Field) recordAST()  {}
func (*Spread) recordAST() {}

func (*Spread) vectorElem()      {}
func (*VectorValue) vectorElem() {}

// Various Expr fields.

type (
	Field struct {
		Kind  string `json:"kind" unpack:""`
		Name  string `json:"name"`
		Value Expr   `json:"value"`
	}
	Spread struct {
		Kind string `json:"kind" unpack:""`
		Expr Expr   `json:"expr"`
	}
	Entry struct {
		Key   Expr `json:"key"`
		Value Expr `json:"value"`
	}
	VectorValue struct {
		Kind string `json:"kind" unpack:""`
		Expr Expr   `json:"expr"`
	}
)

func (*Agg) ExprDAG()          {}
func (*Assignment) ExprDAG()   {}
func (*ArrayExpr) ExprDAG()    {}
func (*BinaryExpr) ExprDAG()   {}
func (*Call) ExprDAG()         {}
func (*Conditional) ExprDAG()  {}
func (*Dot) ExprDAG()          {}
func (*Literal) ExprDAG()      {}
func (*MapExpr) ExprDAG()      {}
func (*RecordExpr) ExprDAG()   {}
func (*RegexpMatch) ExprDAG()  {}
func (*RegexpSearch) ExprDAG() {}
func (*Search) ExprDAG()       {}
func (*SetExpr) ExprDAG()      {}
func (*This) ExprDAG()         {}
func (*UnaryExpr) ExprDAG()    {}
func (*Var) ExprDAG()          {}
func (*OverExpr) ExprDAG()     {}

func IsThis(e Expr) bool {
	if p, ok := e.(*This); ok {
		return len(p.Path) == 0
	}
	return false
}

func TopLevelField(e Expr) (string, bool) {
	if b, ok := e.(*This); ok {
		if len(b.Path) == 1 {
			return b.Path[0], true
		}
	}
	return "", false
}

func IsTopLevelField(e Expr) bool {
	_, ok := TopLevelField(e)
	return ok
}

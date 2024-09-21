package ast

// XXX sql declares the types used to represent syntax trees for SuperSQL
// queries which are compiled into the Zed runtime DAG.  We reuse Zed AST nodes
// for types, expressions, etc and only have SQL-specific elements here.

type Select struct {
	Kind     string      `json:"kind" unpack:""`
	Distinct bool        `json:"distinct"`
	Value    bool        `json:"value"`
	Args     Assignments `json:"args"`
	From     []Op        `json:"from"`
	Where    Expr        `json:"where"`
	GroupBy  []Expr      `json:"group_by"`
	Having   Expr        `json:"having"`
}

type Limit struct {
	Kind   string `json:"kind" unpack:""`
	Op     Op
	Count  Expr
	Offset Expr
}

type With struct {
	Kind      string `json:"kind" unpack:""`
	Body      Op
	Recursive bool
	CTEs      []CTE
}

type CTE struct {
	Name         string
	Materialized *bool
	Op           Op
}

type OrderBy struct {
	Kind  string `json:"kind" unpack:""`
	Op    Op
	Exprs []SortExpr
}

// An Op is a node in the flowgraph that takes Zed values in, operates upon them,
// and produces Zed values as output.
type (
	//XXX not using this yet
	CaseExpr struct {
		Expr  Expr
		Whens []When
		Else  Expr
	}
	When struct {
		Cond  Expr
		Value Expr
	}
	SQLJoin struct { //XXX
		Kind  string `json:"kind" unpack:""`
		Style string // "full", "left", "right", "inner"
		Left  Op
		Right Op
		Cond  JoinExpr
	}
	CrossJoin struct {
		Kind  string `json:"kind" unpack:""`
		Left  Op
		Right Op
	}
	Union struct {
		Kind     string `json:"kind" unpack:""`
		Distinct bool
		Left     Op
		Right    Op
	}
)

type JoinExpr interface {
	JoinOp()
}

type JoinOn struct {
	Kind string `json:"kind" unpack:""`
	Expr Expr
}

func (*JoinOn) JoinOp() {}

type JoinUsing struct {
	Kind   string `json:"kind" unpack:""`
	Fields []Expr
}

func (*JoinUsing) JoinOp() {}

// Source structure

type (
	Table struct {
		Kind string `json:"kind" unpack:""`
		Name string `json:"name"`
	}
	Ordinality struct {
		Kind string `json:"kind" unpack:""`
		Op
	}
	Alias struct {
		Kind string `json:"kind" unpack:""`
		Op
		Name string
	}
)

func (*Select) OpAST()     {}
func (*Table) OpAST()      {}
func (*Ordinality) OpAST() {}
func (*Alias) OpAST()      {}
func (*CrossJoin) OpAST()  {}
func (*SQLJoin) OpAST()    {}
func (*Union) OpAST()      {}
func (*OrderBy) OpAST()    {}
func (*Limit) OpAST()      {}
func (*With) OpAST()       {}

func (*Select) Pos() int { return 0 } //XXX
func (*Select) End() int { return 0 } //XXX

func (*Ordinality) Pos() int { return 0 } //XXX
func (*Ordinality) End() int { return 0 } //XXX

func (*Alias) Pos() int { return 0 } //XXX
func (*Alias) End() int { return 0 } //XXX

func (*Table) Pos() int { return 0 } //XXX
func (*Table) End() int { return 0 } //XXX

func (*CrossJoin) Pos() int { return 0 } //XXX
func (*CrossJoin) End() int { return 0 } //XXX

func (*SQLJoin) Pos() int { return 0 } //XXX
func (*SQLJoin) End() int { return 0 } //XXX

func (*Union) Pos() int { return 0 } //XXX
func (*Union) End() int { return 0 } //XXX

func (*OrderBy) Pos() int { return 0 } //XXX
func (*OrderBy) End() int { return 0 } //XXX

func (*Limit) Pos() int { return 0 } //XXX
func (*Limit) End() int { return 0 } //XXX

func (*With) Pos() int { return 0 } //XXX
func (*With) End() int { return 0 } //XXX

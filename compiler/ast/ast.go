// Package ast declares the types used to represent syntax trees for Zed
// queries.
package ast

import (
	astzed "github.com/brimdata/zed/compiler/ast/zed"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/field"
)

// This module is derived from the GO AST design pattern in
// https://golang.org/pkg/go/ast/
//
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Op is the interface implemented by all AST operator nodes.
type Op interface {
	OpAST()
}

type Expr interface {
	ExprAST()
}

type ID struct {
	Kind string `json:"kind" unpack:""`
	Name string `json:"name"`
}

type Term struct {
	Kind  string     `json:"kind" unpack:""`
	Text  string     `json:"text"`
	Value astzed.Any `json:"value"`
}

type UnaryExpr struct {
	Kind    string `json:"kind" unpack:""`
	Op      string `json:"op"`
	Operand Expr   `json:"operand"`
}

// A BinaryExpr is any expression of the form "lhs kind rhs"
// including arithmetic (+, -, *, /), logical operators (and, or),
// comparisons (=, !=, <, <=, >, >=), index operatons (on arrays, sets, and records)
// with kind "[" and a dot expression (".") (on records).
type BinaryExpr struct {
	Kind string `json:"kind" unpack:""`
	Op   string `json:"op"`
	LHS  Expr   `json:"lhs"`
	RHS  Expr   `json:"rhs"`
}

type Conditional struct {
	Kind string `json:"kind" unpack:""`
	Cond Expr   `json:"cond"`
	Then Expr   `json:"then"`
	Else Expr   `json:"else"`
}

// A Call represents different things dependending on its context.
// As a operator, it is either a group-by with no group-by keys and no duration,
// or a filter with a function that is boolean valued.  This is determined
// by the compiler rather than the syntax tree based on the specific functions
// and aggregators that are defined at compile time.  In expression context,
// a function call has the standard semantics where it takes one or more arguments
// and returns a result.
type Call struct {
	Kind  string `json:"kind" unpack:""`
	Name  string `json:"name"`
	Args  []Expr `json:"args"`
	Where Expr   `json:"where"`
}

type Cast struct {
	Kind string `json:"kind" unpack:""`
	Expr Expr   `json:"expr"`
	Type Expr   `json:"type"`
}

type Grep struct {
	Kind    string  `json:"kind" unpack:""`
	Pattern Pattern `json:"pattern"`
	Expr    Expr    `json:"expr"`
}

type Glob struct {
	Kind    string `json:"kind" unpack:""`
	Pattern string `json:"pattern"`
}

type Regexp struct {
	Kind    string `json:"kind" unpack:""`
	Pattern string `json:"pattern"`
}

type String struct {
	Kind string `json:"kind" unpack:""`
	Text string `json:"text"`
}

type Pattern interface {
	PatternAST()
}

func (*Glob) PatternAST()   {}
func (*Regexp) PatternAST() {}
func (*String) PatternAST() {}

type RecordExpr struct {
	Kind  string       `json:"kind" unpack:""`
	Elems []RecordElem `json:"elems"`
}

type RecordElem interface {
	recordAST()
}

func (*Field) recordAST()  {}
func (*ID) recordAST()     {}
func (*Spread) recordAST() {}

type Field struct {
	Kind  string `json:"kind" unpack:""`
	Name  string `json:"name"`
	Value Expr   `json:"value"`
}

type Spread struct {
	Kind string `json:"kind" unpack:""`
	Expr Expr   `json:"expr"`
}

type ArrayExpr struct {
	Kind  string       `json:"kind" unpack:""`
	Elems []VectorElem `json:"elems"`
}

type SetExpr struct {
	Kind  string       `json:"kind" unpack:""`
	Elems []VectorElem `json:"elems"`
}

type VectorElem interface {
	vectorAST()
}

func (*Spread) vectorAST()      {}
func (*VectorValue) vectorAST() {}

type VectorValue struct {
	Kind string `json:"kind" unpack:""`
	Expr Expr   `json:"expr"`
}

type MapExpr struct {
	Kind    string      `json:"kind" unpack:""`
	Entries []EntryExpr `json:"entries"`
}

type EntryExpr struct {
	Key   Expr `json:"key"`
	Value Expr `json:"value"`
}

type OverExpr struct {
	Kind   string      `json:"kind" unpack:""`
	Locals []Def       `json:"locals"`
	Exprs  []Expr      `json:"exprs"`
	Scope  *Sequential `json:"scope"`
}

func (*UnaryExpr) ExprAST()   {}
func (*BinaryExpr) ExprAST()  {}
func (*Conditional) ExprAST() {}
func (*Call) ExprAST()        {}
func (*Cast) ExprAST()        {}
func (*ID) ExprAST()          {}

func (*Assignment) ExprAST() {}
func (*Agg) ExprAST()        {}
func (*Grep) ExprAST()       {}
func (*Glob) ExprAST()       {}
func (*Regexp) ExprAST()     {}
func (*Term) ExprAST()       {}

func (*RecordExpr) ExprAST() {}
func (*ArrayExpr) ExprAST()  {}
func (*SetExpr) ExprAST()    {}
func (*MapExpr) ExprAST()    {}

func (*OverExpr) ExprAST() {}

func (*SQLExpr) ExprAST() {}

// ----------------------------------------------------------------------------
// Operators

// An Op is a node in the flowgraph that takes Zed values in, operates upon them,
// and produces Zed values as output.
type (
	// A Sequential operator represents a set of operators that receive
	// a stream of Zed values from their parent into the first operator
	// and each subsequent operator processes the output records from the
	// previous operator.
	Sequential struct {
		Kind   string `json:"kind" unpack:""`
		Consts []Def  `json:"consts"`
		Ops    []Op   `json:"ops"`
	}
	// A Parallel operator represents a set of operators that each get
	// a stream of Zed values from their parent.
	Parallel struct {
		Kind string `json:"kind" unpack:""`
		// If non-zero, MergeBy contains the field name on
		// which the branches of this parallel operator should be
		// merged in the order indicated by MergeReverse.
		// XXX merge_by should be a list of expressions
		MergeBy      field.Path `json:"merge_by,omitempty"`
		MergeReverse bool       `json:"merge_reverse,omitempty"`
		Ops          []Op       `json:"ops"`
	}
	Switch struct {
		Kind  string `json:"kind" unpack:""`
		Expr  Expr   `json:"expr"`
		Cases []Case `json:"cases"`
	}
	Sort struct {
		Kind       string      `json:"kind" unpack:""`
		Args       []Expr      `json:"args"`
		Order      order.Which `json:"order"`
		NullsFirst bool        `json:"nullsfirst"`
	}
	Cut struct {
		Kind string       `json:"kind" unpack:""`
		Args []Assignment `json:"args"`
	}
	Drop struct {
		Kind string `json:"kind" unpack:""`
		Args []Expr `json:"args"`
	}
	Head struct {
		Kind  string `json:"kind" unpack:""`
		Count int    `json:"count"`
	}
	Tail struct {
		Kind  string `json:"kind" unpack:""`
		Count int    `json:"count"`
	}
	Pass struct {
		Kind string `json:"kind" unpack:""`
	}
	Uniq struct {
		Kind  string `json:"kind" unpack:""`
		Cflag bool   `json:"cflag"`
	}
	Summarize struct {
		Kind  string       `json:"kind" unpack:""`
		Limit int          `json:"limit"`
		Keys  []Assignment `json:"keys"`
		Aggs  []Assignment `json:"aggs"`
	}
	Top struct {
		Kind  string `json:"kind" unpack:""`
		Limit int    `json:"limit"`
		Args  []Expr `json:"args"`
		Flush bool   `json:"flush"`
	}
	Put struct {
		Kind string       `json:"kind" unpack:""`
		Args []Assignment `json:"args"`
	}
	Merge struct {
		Kind string `json:"kind" unpack:""`
		Expr Expr   `json:"expr"`
	}
	Over struct {
		Kind  string      `json:"kind" unpack:""`
		Exprs []Expr      `json:"exprs"`
		Scope *Sequential `json:"scope"`
	}
	Let struct {
		Kind   string `json:"kind" unpack:""`
		Locals []Def  `json:"locals"`
		Over   *Over  `json:"over"`
	}
	Search struct {
		Kind string `json:"kind" unpack:""`
		Expr Expr   `json:"expr"`
	}
	Where struct {
		Kind string `json:"kind" unpack:""`
		Expr Expr   `json:"expr"`
	}
	Yield struct {
		Kind  string `json:"kind" unpack:""`
		Exprs []Expr `json:"exprs"`
	}
	// An OpAssignment is a list of assignments whose parent operator
	// is unknown: It could be a Summarize or Put operator. This will be
	// determined in the semantic phase.
	OpAssignment struct {
		Kind        string       `json:"kind" unpack:""`
		Assignments []Assignment `json:"assignments"`
	}
	// An OpExpr operator is an expression that appears as an operator
	// and requires semantic analysis to determine if it is a filter, a yield,
	// or an aggregation.
	OpExpr struct {
		Kind string `json:"kind" unpack:""`
		Expr Expr   `json:"expr"`
	}
	Rename struct {
		Kind string       `json:"kind" unpack:""`
		Args []Assignment `json:"args"`
	}
	Fuse struct {
		Kind string `json:"kind" unpack:""`
	}
	Join struct {
		Kind     string       `json:"kind" unpack:""`
		Style    string       `json:"style"`
		LeftKey  Expr         `json:"left_key"`
		RightKey Expr         `json:"right_key"`
		Args     []Assignment `json:"args"`
	}
	// A SQLExpr can be an operator, an expression inside of a SQL FROM clause,
	// or an expression used as a Zed value generator.  Currenly, the "select"
	// keyword collides with the select() generator function (it can be parsed
	// unambiguosly because of the parens but this is not user friendly
	// so we need a new name for select()... see issue #2133).
	// TBD: from alias, "in" over tuples, WITH sub-queries, multi-table FROM
	// implying a JOIN, aliases for tables in FROM and JOIN.
	SQLExpr struct {
		Kind    string       `json:"kind" unpack:""`
		Select  []Assignment `json:"select"`
		From    *SQLFrom     `json:"from"`
		Joins   []SQLJoin    `json:"joins"`
		Where   Expr         `json:"where"`
		GroupBy []Expr       `json:"group_by"`
		Having  Expr         `json:"having"`
		OrderBy *SQLOrderBy  `json:"order_by"`
		Limit   int          `json:"limit"`
	}
	Shape struct {
		Kind string `json:"kind" unpack:""`
	}
	From struct {
		Kind   string  `json:"kind" unpack:""`
		Trunks []Trunk `json:"trunks"`
	}
)

// Source structure

type (
	File struct {
		Kind   string  `json:"kind" unpack:""`
		Path   string  `json:"path"`
		Format string  `json:"format"`
		Layout *Layout `json:"layout"`
	}
	HTTP struct {
		Kind   string  `json:"kind" unpack:""`
		URL    string  `json:"url"`
		Format string  `json:"format"`
		Layout *Layout `json:"layout"`
	}
	Pool struct {
		Kind      string   `json:"kind" unpack:""`
		Spec      PoolSpec `json:"spec"`
		At        string   `json:"at"`
		ScanOrder string   `json:"scan_order"` // asc, desc, or unknown
	}
	Explode struct {
		Kind string      `json:"kind" unpack:""`
		Args []Expr      `json:"args"`
		Type astzed.Type `json:"type"`
		As   Expr        `json:"as"`
	}
)

type PoolSpec struct {
	Pool   string `json:"pool"`
	Commit string `json:"commit"`
	Meta   string `json:"meta"`
}

type Source interface {
	Source()
}

func (*Pool) Source() {}
func (*File) Source() {}
func (*HTTP) Source() {}
func (*Pass) Source() {}

type Layout struct {
	Kind  string `json:"kind" unpack:""`
	Keys  []Expr `json:"keys"`
	Order string `json:"order"`
}

type Trunk struct {
	Kind   string      `json:"kind" unpack:""`
	Source Source      `json:"source"`
	Seq    *Sequential `json:"seq"`
}

type Case struct {
	Expr Expr `json:"expr"`
	Op   Op   `json:"op"`
}

type SQLFrom struct {
	Table Expr `json:"table"`
	Alias Expr `json:"alias"`
}

type SQLOrderBy struct {
	Kind  string      `json:"kind" unpack:""`
	Keys  []Expr      `json:"keys"`
	Order order.Which `json:"order"`
}

type SQLJoin struct {
	Table    Expr   `json:"table"`
	Style    string `json:"style"`
	LeftKey  Expr   `json:"left_key"`
	RightKey Expr   `json:"right_key"`
	Alias    Expr   `json:"alias"`
}

type Assignment struct {
	Kind string `json:"kind" unpack:""`
	LHS  Expr   `json:"lhs"`
	RHS  Expr   `json:"rhs"`
}

// Def is like Assignment but the LHS is an identifier that may be later
// referenced.  This is used for const blocks in Sequential and var blocks
// in a let scope.
type Def struct {
	Name string `json:"name"`
	Expr Expr   `json:"expr"`
}

func (*Sequential) OpAST()   {}
func (*Parallel) OpAST()     {}
func (*Switch) OpAST()       {}
func (*Sort) OpAST()         {}
func (*Cut) OpAST()          {}
func (*Drop) OpAST()         {}
func (*Head) OpAST()         {}
func (*Tail) OpAST()         {}
func (*Pass) OpAST()         {}
func (*Uniq) OpAST()         {}
func (*Summarize) OpAST()    {}
func (*Top) OpAST()          {}
func (*Put) OpAST()          {}
func (*OpAssignment) OpAST() {}
func (*OpExpr) OpAST()       {}
func (*Rename) OpAST()       {}
func (*Fuse) OpAST()         {}
func (*Join) OpAST()         {}
func (*Shape) OpAST()        {}
func (*From) OpAST()         {}
func (*Explode) OpAST()      {}
func (*Merge) OpAST()        {}
func (*Over) OpAST()         {}
func (*Let) OpAST()          {}
func (*Search) OpAST()       {}
func (*Where) OpAST()        {}
func (*Yield) OpAST()        {}

func (*SQLExpr) OpAST() {}

func (seq *Sequential) Prepend(front Op) {
	seq.Ops = append([]Op{front}, seq.Ops...)
}

// An Agg is an AST node that represents a aggregate function.  The Name
// field indicates the aggregation method while the Expr field indicates
// an expression applied to the incoming records that is operated upon by them
// aggregate function.  If Expr isn't present, then the aggregator doesn't act
// upon a function of the record, e.g., count() counts up records without
// looking into them.
type Agg struct {
	Kind  string `json:"kind" unpack:""`
	Name  string `json:"name"`
	Expr  Expr   `json:"expr"`
	Where Expr   `json:"where"`
}

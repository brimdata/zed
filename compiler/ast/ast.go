// Package ast declares the types used to represent syntax trees for Zed
// queries.
package ast

import (
	"github.com/brimdata/zed/compiler/ast/node"
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
	node.Node
	OpAST()
}

type Decl interface {
	node.Node
	DeclAST()
}

type Expr interface {
	node.Node
	ExprAST()
}

type ID struct {
	node.Base
	Kind string `json:"kind" unpack:""`
	Name string `json:"name"`
}

type Term struct {
	node.Base
	Kind  string     `json:"kind" unpack:""`
	Text  string     `json:"text"`
	Value astzed.Any `json:"value"`
}

type UnaryExpr struct {
	node.Base
	Kind    string `json:"kind" unpack:""`
	Op      string `json:"op"`
	Operand Expr   `json:"operand"`
}

// A BinaryExpr is any expression of the form "lhs kind rhs"
// including arithmetic (+, -, *, /), logical operators (and, or),
// comparisons (=, !=, <, <=, >, >=), index operatons (on arrays, sets, and records)
// with kind "[" and a dot expression (".") (on records).
type BinaryExpr struct {
	node.Base
	Kind string `json:"kind" unpack:""`
	Op   string `json:"op"`
	LHS  Expr   `json:"lhs"`
	RHS  Expr   `json:"rhs"`
}

func (b *BinaryExpr) Pos() (int, int) {
}

type Conditional struct {
	node.Base
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
	node.Base
	Kind  string `json:"kind" unpack:""`
	Name  string `json:"name"`
	Args  []Expr `json:"args"`
	Where Expr   `json:"where"`
}

type Cast struct {
	node.Base
	Kind string `json:"kind" unpack:""`
	Expr Expr   `json:"expr"`
	Type Expr   `json:"type"`
}

type Grep struct {
	node.Base
	Kind    string `json:"kind" unpack:""`
	Pattern Expr   `json:"pattern"`
	Expr    Expr   `json:"expr"`
}

type Glob struct {
	node.Base
	Kind    string `json:"kind" unpack:""`
	Pattern string `json:"pattern"`
}

type QuotedString struct {
	Kind string `json:"kind" unpack:""`
	Text string `json:"text"`
}

type Regexp struct {
	node.Base
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

func (*Glob) PatternAST()         {}
func (*QuotedString) PatternAST() {}
func (*Regexp) PatternAST()       {}
func (*String) PatternAST()       {}

type RecordExpr struct {
	node.Base
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
	node.Base
	Kind  string `json:"kind" unpack:""`
	Name  string `json:"name"`
	Value Expr   `json:"value"`
}

type Spread struct {
	node.Base
	Kind string `json:"kind" unpack:""`
	Expr Expr   `json:"expr"`
}

type ArrayExpr struct {
	node.Base
	Kind  string       `json:"kind" unpack:""`
	Elems []VectorElem `json:"elems"`
}

type SetExpr struct {
	node.Base
	Kind  string       `json:"kind" unpack:""`
	Elems []VectorElem `json:"elems"`
}

type VectorElem interface {
	vectorAST()
}

func (*Spread) vectorAST()      {}
func (*VectorValue) vectorAST() {}

type VectorValue struct {
	node.Base
	Kind string `json:"kind" unpack:""`
	Expr Expr   `json:"expr"`
}

type MapExpr struct {
	node.Base
	Kind    string      `json:"kind" unpack:""`
	Entries []EntryExpr `json:"entries"`
}

type EntryExpr struct {
	node.Base
	Key   Expr `json:"key"`
	Value Expr `json:"value"`
}

type OverExpr struct {
	node.Base
	Kind   string `json:"kind" unpack:""`
	Locals []Def  `json:"locals"`
	Exprs  []Expr `json:"exprs"`
	Body   Seq    `json:"body"`
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

type ConstDecl struct {
	node.Base
	Kind string `json:"kind" unpack:""`
	Name string `json:"name"`
	Expr Expr   `json:"expr"`
}

type FuncDecl struct {
	node.Base
	Kind   string   `json:"kind" unpack:""`
	Name   string   `json:"name"`
	Params []string `json:"params"`
	Expr   Expr     `json:"expr"`
}

type OpDecl struct {
	node.Base
	Kind   string   `json:"kind" unpack:""`
	Name   string   `json:"name"`
	Params []string `json:"params"`
	Body   Seq      `json:"body"`
}

func (*ConstDecl) DeclAST() {}
func (*FuncDecl) DeclAST()  {}
func (*OpDecl) DeclAST()    {}

// ----------------------------------------------------------------------------
// Operators

// A Seq represents a sequence of operators that receive
// a stream of Zed values from their parent into the first operator
// and each subsequent operator processes the output records from the
// previous operator.
type Seq []Op

// An Op is a node in the flowgraph that takes Zed values in, operates upon them,
// and produces Zed values as output.
type (
	Scope struct {
		node.Base
		Kind  string `json:"kind" unpack:""`
		Decls []Decl `json:"decls"`
		Body  Seq    `json:"body"`
	}
	// A Parallel operator represents a set of operators that each get
	// a stream of Zed values from their parent.
	Parallel struct {
		node.Base
		Kind string `json:"kind" unpack:""`
		// If non-zero, MergeBy contains the field name on
		// which the branches of this parallel operator should be
		// merged in the order indicated by MergeReverse.
		// XXX merge_by should be a list of expressions
		MergeBy      field.Path `json:"merge_by,omitempty"`
		MergeReverse bool       `json:"merge_reverse,omitempty"`
		Paths        []Seq      `json:"paths"`
	}
	Switch struct {
		node.Base
		Kind  string `json:"kind" unpack:""`
		Expr  Expr   `json:"expr"`
		Cases []Case `json:"cases"`
	}
	Sort struct {
		node.Base
		Kind       string      `json:"kind" unpack:""`
		Args       []Expr      `json:"args"`
		Order      order.Which `json:"order"`
		NullsFirst bool        `json:"nullsfirst"`
	}
	Cut struct {
		node.Base
		Kind string       `json:"kind" unpack:""`
		Args []Assignment `json:"args"`
	}
	Drop struct {
		node.Base
		Kind string `json:"kind" unpack:""`
		Args []Expr `json:"args"`
	}
	Explode struct {
		node.Base
		Kind string      `json:"kind" unpack:""`
		Args []Expr      `json:"args"`
		Type astzed.Type `json:"type"`
		As   Expr        `json:"as"`
	}
	Head struct {
		node.Base
		Kind  string `json:"kind" unpack:""`
		Count Expr   `json:"count"`
	}
	Tail struct {
		node.Base
		Kind  string `json:"kind" unpack:""`
		Count Expr   `json:"count"`
	}
	Pass struct {
		node.Base
		Kind string `json:"kind" unpack:""`
	}
	Uniq struct {
		node.Base
		Kind  string `json:"kind" unpack:""`
		Cflag bool   `json:"cflag"`
	}
	Summarize struct {
		node.Base
		Kind  string       `json:"kind" unpack:""`
		Limit int          `json:"limit"`
		Keys  []Assignment `json:"keys"`
		Aggs  []Assignment `json:"aggs"`
	}
	Top struct {
		node.Base
		Kind  string `json:"kind" unpack:""`
		Limit int    `json:"limit"`
		Args  []Expr `json:"args"`
		Flush bool   `json:"flush"`
	}
	Put struct {
		node.Base
		Kind string       `json:"kind" unpack:""`
		Args []Assignment `json:"args"`
	}
	Merge struct {
		node.Base
		Kind string `json:"kind" unpack:""`
		Expr Expr   `json:"expr"`
	}
	Over struct {
		node.Base
		Kind   string `json:"kind" unpack:""`
		Exprs  []Expr `json:"exprs"`
		Locals []Def  `json:"locals"`
		Body   Seq    `json:"body"`
	}
	Search struct {
		node.Base
		Kind string `json:"kind" unpack:""`
		Expr Expr   `json:"expr"`
	}
	Where struct {
		node.Base
		Kind string `json:"kind" unpack:""`
		Expr Expr   `json:"expr"`
	}
	Yield struct {
		node.Base
		Kind  string `json:"kind" unpack:""`
		Exprs []Expr `json:"exprs"`
	}
	// An OpAssignment is a list of assignments whose parent operator
	// is unknown: It could be a Summarize or Put operator. This will be
	// determined in the semantic phase.
	OpAssignment struct {
		node.Base
		Kind        string       `json:"kind" unpack:""`
		Assignments []Assignment `json:"assignments"`
	}
	// An OpExpr operator is an expression that appears as an operator
	// and requires semantic analysis to determine if it is a filter, a yield,
	// or an aggregation.
	OpExpr struct {
		node.Base
		Kind string `json:"kind" unpack:""`
		Expr Expr   `json:"expr"`
	}
	Rename struct {
		node.Base
		Kind string       `json:"kind" unpack:""`
		Args []Assignment `json:"args"`
	}
	Fuse struct {
		node.Base
		Kind string `json:"kind" unpack:""`
	}
	Join struct {
		node.Base
		Kind       string       `json:"kind" unpack:""`
		Style      string       `json:"style"`
		RightInput Seq          `json:"right_input"`
		LeftKey    Expr         `json:"left_key"`
		RightKey   Expr         `json:"right_key"`
		Args       []Assignment `json:"args"`
	}
	Sample struct {
		node.Base
		Kind string `json:"kind" unpack:""`
		Expr Expr   `json:"expr"`
	}
	// A SQLExpr can be an operator, an expression inside of a SQL FROM clause,
	// or an expression used as a Zed value generator.  Currenly, the "select"
	// keyword collides with the select() generator function (it can be parsed
	// unambiguosly because of the parens but this is not user friendly
	// so we need a new name for select()... see issue #2133).
	// TBD: from alias, "in" over tuples, WITH sub-queries, multi-table FROM
	// implying a JOIN, aliases for tables in FROM and JOIN.
	SQLExpr struct {
		node.Base
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
		node.Base
		Kind string `json:"kind" unpack:""`
	}
	From struct {
		node.Base
		Kind   string  `json:"kind" unpack:""`
		Trunks []Trunk `json:"trunks"`
	}
	Load struct {
		node.Base
		Kind    string `json:"kind" unpack:""`
		Pool    string `json:"pool"`
		Branch  string `json:"branch"`
		Author  string `json:"author"`
		Message string `json:"message"`
		Meta    string `json:"meta"`
	}
)

// Source structure

type (
	File struct {
		node.Base
		Kind    string   `json:"kind" unpack:""`
		Path    Pattern  `json:"path"`
		Format  string   `json:"format"`
		SortKey *SortKey `json:"sort_key"`
	}
	HTTP struct {
		node.Base
		Kind    string      `json:"kind" unpack:""`
		URL     Pattern     `json:"url"`
		Format  string      `json:"format"`
		SortKey *SortKey    `json:"sort_key"`
		Method  string      `json:"method"`
		Headers *RecordExpr `json:"headers"`
		Body    string      `json:"body"`
	}
	Pool struct {
		node.Base
		Kind   string   `json:"kind" unpack:""`
		Spec   PoolSpec `json:"spec"`
		At     string   `json:"at"`
		Delete bool     `json:"delete"`
	}
)

type PoolSpec struct {
	node.Base
	Pool   Pattern `json:"pool"`
	Commit string  `json:"commit"`
	Meta   string  `json:"meta"`
	Tap    bool    `json:"tap"`
}

type Source interface {
	node.Node
	Source()
}

func (*Pool) Source() {}
func (*File) Source() {}
func (*HTTP) Source() {}
func (*Pass) Source() {}

type SortKey struct {
	Kind  string `json:"kind" unpack:""`
	Keys  []Expr `json:"keys"`
	Order string `json:"order"`
}

type Trunk struct {
	node.Base
	Kind   string `json:"kind" unpack:""`
	Source Source `json:"source"`
	Seq    Seq    `json:"seq"`
}

type Case struct {
	node.Base
	Expr Expr `json:"expr"`
	Path Seq  `json:"path"`
}

type SQLFrom struct {
	node.Base
	Table Expr `json:"table"`
	Alias Expr `json:"alias"`
}

type SQLOrderBy struct {
	node.Base
	Kind  string      `json:"kind" unpack:""`
	Keys  []Expr      `json:"keys"`
	Order order.Which `json:"order"`
}

type SQLJoin struct {
	node.Base
	Table    Expr   `json:"table"`
	Style    string `json:"style"`
	LeftKey  Expr   `json:"left_key"`
	RightKey Expr   `json:"right_key"`
	Alias    Expr   `json:"alias"`
}

type Assignment struct {
	node.Base
	Kind string `json:"kind" unpack:""`
	LHS  Expr   `json:"lhs"`
	RHS  Expr   `json:"rhs"`
}

// Def is like Assignment but the LHS is an identifier that may be later
// referenced.  This is used for const blocks in Sequential and var blocks
// in a let scope.
type Def struct {
	node.Base
	Name string `json:"name"`
	Expr Expr   `json:"expr"`
}

func (*Scope) OpAST()        {}
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
func (*Search) OpAST()       {}
func (*Where) OpAST()        {}
func (*Yield) OpAST()        {}
func (*Sample) OpAST()       {}
func (*Load) OpAST()         {}

func (*SQLExpr) OpAST() {}

func (seq *Seq) Prepend(front Op) {
	*seq = append([]Op{front}, *seq...)
}

// An Agg is an AST node that represents a aggregate function.  The Name
// field indicates the aggregation method while the Expr field indicates
// an expression applied to the incoming records that is operated upon by them
// aggregate function.  If Expr isn't present, then the aggregator doesn't act
// upon a function of the record, e.g., count() counts up records without
// looking into them.
type Agg struct {
	node.Base
	Kind  string `json:"kind" unpack:""`
	Name  string `json:"name"`
	Expr  Expr   `json:"expr"`
	Where Expr   `json:"where"`
}

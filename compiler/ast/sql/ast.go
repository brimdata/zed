// Package sql declares the types used to represent syntax trees for SuperSQL
// queries which are compiled into the Zed runtime DAG.  We reuse Zed AST nodes
// for types, expressions, etc and only have SQL-specific elements here.
package sql

import (
	"github.com/brimdata/zed/compiler/ast"
)

// This module is derived from the GO AST design pattern in
// https://golang.org/pkg/go/ast/
//
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

type Node interface {
	Pos() int // Position of first character belonging to the node.
	End() int // Position of first character immediately after the node.
}

// Op is the interface implemented by all AST operator nodes.
type Op interface {
	Node
	OpAST()
}

type Select struct {
	Distinct bool
	Exprs    []ast.Expr
	From     []Source
	Where    ast.Expr
	GroupBy  GroupBy
	Having   ast.Expr
	OrderBy  []Order
	Limit    ast.Expr
}

//XXX need to handle named tables

type Lateral struct {
	Source
}

type GroupBy struct {
	Keys ast.Assignments
	Aggs ast.Assignments
}

type Order struct {
	Expr       ast.Expr
	Order      string // asc or desc
	NullsFirst bool
}

// An Op is a node in the flowgraph that takes Zed values in, operates upon them,
// and produces Zed values as output.
type (
	CaseExpr struct {
		Expr  ast.Expr
		Whens []*When
		Else  ast.Expr
	}
	When struct {
		Cond  ast.Expr
		Value ast.Expr
	}
	Limit struct {
		Count ast.Expr
	}
	//XXX keep this for alternative to SQL++ left correlations
	// maybe we reuse unnest terminology?
	Over struct {
		Exprs  []ast.Expr
		Locals []Def
		Body   Select
	}
	Search struct {
		Expr ast.Expr
	}
	Where struct {
		Expr ast.Expr `json:"expr"`
	}
	JoinTableExpr struct {
		Left  Source
		Join  string
		Right Source
		On    ast.Expr
	}
	Union struct {
		Type        string
		Left, Right Select
		OrderBy     []Order
		Limit       *Limit
	}
)

// Source structure

type (
	File struct {
		Path    ast.Pattern
		Format  string
		SortKey *ast.SortExpr
		EndPos  int
	}
	HTTP struct {
		URL     ast.Pattern
		Format  string
		SortKey *ast.SortExpr
		Method  string
		Headers *ast.RecordExpr
		Body    string
		EndPos  int
	}
	Pool struct {
		Spec ast.PoolSpec
	}
	Table struct {
		Name string
	}
	Alias struct {
		Source
		Name string
	}
	Ordinality struct {
		Source
	}
)

type PoolSpec struct {
	Pool   ast.Pattern
	Commit string
	Meta   string
	Tap    bool
}

// XXX Source implements different operators that return
// dynamic data sets. XXX change name

type Source interface {
	Node
	DDS()
}

func (*Pool) DDS()       {}
func (*File) DDS()       {}
func (*HTTP) DDS()       {}
func (*Select) DDS()     {}
func (*Table) DDS()      {}
func (*Alias) DDS()      {}
func (*Ordinality) DDS() {}

// Def is like Assignment but the LHS is an identifier that may be later
// referenced.  This is used for const blocks in Sequential and var blocks
// in a let scope.
type Def struct {
	Name *ast.ID  `json:"name"`
	Expr ast.Expr `json:"expr"`
}

func (*Pool) OpAST()          {}
func (*File) OpAST()          {}
func (*HTTP) OpAST()          {}
func (*GroupBy) OpAST()       {}
func (*JoinTableExpr) OpAST() {}
func (*Over) OpAST()          {}
func (*Search) OpAST()        {}
func (*Where) OpAST()         {}

// An Agg is an AST node that represents a aggregate function.  The Name
// field indicates the aggregation method while the Expr field indicates
// an expression applied to the incoming records that is operated upon by them
// aggregate function.  If Expr isn't present, then the aggregator doesn't act
// upon a function of the record, e.g., count() counts up records without
// looking into them.
type Agg struct {
	Name  string
	Expr  ast.Expr
	Where ast.Expr
}

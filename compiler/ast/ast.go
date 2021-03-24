// Package ast declares the types used to represent syntax trees for zql
// queries.
package ast

// This module is derived from the GO ast design pattern
// https://golang.org/pkg/go/ast/
//
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import (
	"github.com/brimsec/zq/field"
)

// Proc is the interface implemented by all AST processor nodes.
type Proc interface {
	ProcNode()
}

// Id refers to a syntax element analogous to a programming language identifier.
type Id struct {
	Kind string `json:"kind" unpack:""`
	Name string `json:"name"`
}

type Path struct {
	Kind string   `json:"kind" unpack:""`
	Name []string `json:"name"`
}

type Ref struct {
	Kind string `json:"kind" unpack:""`
	Name string `json:"name"`
}

// Root refers to the outer record being operated upon.  Field accesses
// typically begin with the LHS of a "." expression set to a Root.
type Root struct {
	Kind string `json:"kind" unpack:""`
}

type Expr interface {
	exprNode()
}

type Search struct {
	Kind  string    `json:"kind" unpack:""`
	Text  string    `json:"text"`
	Value Primitive `json:"value"` //XXX search should be extended to complex types
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

type SelectExpr struct {
	Kind      string `json:"kind" unpack:""`
	Selectors []Expr `json:"selectors"`
	Methods   []Call `json:"methods"`
}

type Conditional struct {
	Kind string `json:"kind" unpack:""`
	Cond Expr   `json:"cond"`
	Then Expr   `json:"then"`
	Else Expr   `json:"else"`
}

// A Call represents different things dependending on its context.
// As a proc, it is either a group-by with no group-by keys and no duration,
// or a filter with a function that is boolean valued.  This is determined
// by the compiler rather than the syntax tree based on the specific functions
// and aggregators that are defined at compile time.  In expression context,
// a function call has the standard semantics where it takes one or more arguments
// and returns a result.
type Call struct {
	Kind string `json:"kind" unpack:""`
	Name string `json:"name"`
	Args []Expr `json:"args"`
}

type Cast struct {
	Kind string `json:"kind" unpack:""`
	Expr Expr   `json:"expr"`
	Type Type   `json:"type"`
}

type SeqExpr struct {
	Kind      string   `json:"kind" unpack:""`
	Name      string   `json:"name"`
	Selectors []Expr   `json:"selectors"`
	Methods   []Method `json:"methods"`
}

type RegexpMatch struct {
	Kind    string `json:"kind" unpack:""`
	Pattern string `json:"pattern"`
	Expr    Expr   `json:"expr"`
}

type RegexpSearch struct {
	Kind    string `json:"kind" unpack:""`
	Pattern string `json:"pattern"`
}

type RecordExpr struct {
	Kind   string      `json:"kind" unpack:""`
	Fields []FieldExpr `json:"fields"`
}

type FieldExpr struct {
	Name  string `json:"name"`
	Value Expr   `json:"value"`
}

type ArrayExpr struct {
	Kind  string `json:"kind" unpack:""`
	Exprs []Expr `json:"exprs"`
}

type SetExpr struct {
	Kind  string `json:"kind" unpack:""`
	Exprs []Expr `json:"exprs"`
}

type MapExpr struct {
	Kind    string      `json:"kind" unpack:""`
	Entries []EntryExpr `json:"entries"`
}

type EntryExpr struct {
	Key   Expr `json:"key"`
	Value Expr `json:"value"`
}

func (*UnaryExpr) exprNode()   {}
func (*BinaryExpr) exprNode()  {}
func (*SelectExpr) exprNode()  {}
func (*Conditional) exprNode() {}
func (*Search) exprNode()      {}
func (*Call) exprNode()        {}
func (*Cast) exprNode()        {}
func (*Primitive) exprNode()   {}
func (*Id) exprNode()          {}
func (*Path) exprNode()        {}
func (*Ref) exprNode()         {}
func (*Root) exprNode()        {}

func (*Assignment) exprNode()   {}
func (*Agg) exprNode()          {}
func (*SeqExpr) exprNode()      {}
func (*RegexpSearch) exprNode() {}
func (*RegexpMatch) exprNode()  {}
func (*TypeValue) exprNode()    {}

func (*RecordExpr) exprNode() {}
func (*ArrayExpr) exprNode()  {}
func (*SetExpr) exprNode()    {}
func (*MapExpr) exprNode()    {}

func (*SQLExpr) exprNode() {}

// ----------------------------------------------------------------------------
// Procs

// A proc is a node in the flowgraph that takes records in, processes them,
// and produces records as output.
//
type (
	// A Sequential proc represents a set of procs that receive
	// a stream of records from their parent into the first proc
	// and each subsequent proc processes the output records from the
	// previous proc.
	Sequential struct {
		Kind  string `json:"kind" unpack:""`
		Procs []Proc `json:"procs"`
	}
	// A Parallel proc represents a set of procs that each get
	// a stream of records from their parent.
	Parallel struct {
		Kind string `json:"kind" unpack:""`
		// If non-zero, MergeBy contains the field name on
		// which the branches of this parallel proc should be
		// merged in the order indicated by MergeReverse.
		// XXX merge_by should be a list of expressions
		MergeBy      field.Static `json:"merge_by,omitempty"`
		MergeReverse bool         `json:"merge_reverse,omitempty"`
		Procs        []Proc       `json:"procs"`
	}
	// A Switch proc represents a set of procs that each get
	// a stream of records from their parent.
	Switch struct {
		Kind  string `json:"kind" unpack:""`
		Cases []Case `json:"cases"`
		// If non-zero, MergeField contains the field name on
		// which the branches of this parallel proc should be
		// merged in the order indicated by MergeReverse.
		MergeBy      field.Static `json:"merge_by,omitempty"`
		MergeReverse bool         `json:"merge_reverse,omitempty"`
	}
	// A Sort proc represents a proc that sorts records.
	Sort struct {
		Kind       string `json:"kind" unpack:""`
		Args       []Expr `json:"args"`
		SortDir    int    `json:"sortdir"`
		NullsFirst bool   `json:"nullsfirst"`
	}
	// A Cut proc represents a proc that removes fields from each
	// input record where each removed field matches one of the named fields
	// sending each such modified record to its output in the order received.
	Cut struct {
		Kind string       `json:"kind" unpack:""`
		Args []Assignment `json:"args"`
	}
	// A Pick proc is like a Cut but skips records that do not
	// match all of the field expressions.
	Pick struct {
		Kind string       `json:"kind" unpack:""`
		Args []Assignment `json:"args"`
	}
	// A Drop proc represents a proc that removes fields from each
	// input record.
	Drop struct {
		Kind string `json:"kind" unpack:""`
		Args []Expr `json:"args"`
	}
	// A Head proc represents a proc that forwards the indicated number
	// of records then terminates.
	Head struct {
		Kind  string `json:"kind" unpack:""`
		Count int    `json:"count"`
	}
	// A Tail proc represents a proc that reads all its records from its
	// input transmits the final number of records indicated by the count.
	Tail struct {
		Kind  string `json:"kind" unpack:""`
		Count int    `json:"count"`
	}
	// A Filter proc represents a proc that discards all records that do
	// not match the indicfated filter and forwards all that match to its output.
	Filter struct {
		Kind string `json:"kind" unpack:""`
		Expr Expr   `json:"expr"`
	}
	// A Pass proc represents a passthrough proc that mirrors
	// incoming Pull()s on its parent and returns the result.
	Pass struct {
		Kind string `json:"kind" unpack:""`
	}
	// A Uniq proc represents a proc that discards any record that matches
	// the previous record transmitted.  The Cflag causes the output records
	// to contain a new field called count that contains the number of matched
	// records in that set, similar to the unix shell command uniq.
	Uniq struct {
		Kind  string `json:"kind" unpack:""`
		Cflag bool   `json:"cflag"`
	}
	// A Summarize proc represents a proc that consumes all the records
	// in its input, partitions the records into groups based on the values
	// of the fields specified in the keys field (where the first key is the
	// primary grouping key), and applies aggregators (if any) to each group. If the
	// Duration field is non-zero, then the groups are further partioned by time
	// into bins of the duration.  In this case, the primary grouping key is ts.
	// The InputSortDir field indicates that input is sorted (with
	// direction indicated by the sign of the field) in the primary
	// grouping key. In this case, the proc outputs the aggregation
	// results from each key as they complete so that large inputs
	// are processed and streamed efficiently.
	// The Limit field specifies the number of different groups that can be
	// aggregated over. When absent, the runtime defaults to an
	// appropriate value.
	// If PartialsOut is true, the proc will produce partial aggregation
	// output result; likewise, if PartialsIn is true, the proc will
	// expect partial results as input.
	Summarize struct {
		Kind         string       `json:"kind" unpack:""`
		Duration     *Primitive   `json:"duration"`
		InputSortDir int          `json:"input_sort_dir,omitempty"`
		Limit        int          `json:"limit"`
		Keys         []Assignment `json:"keys"`
		Aggs         []Assignment `json:"aggs"`
		PartialsIn   bool         `json:"partials_in,omitempty"`
		PartialsOut  bool         `json:"partials_out,omitempty"`
	}
	// A Top proc is similar to a Sort with a few key differences:
	// - It only sorts in descending order.
	// - It utilizes a MaxHeap, immediately discarding records that are not in
	// the top N of the sort.
	// - It has an option (Flush) to sort and emit on every batch.
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

	// A Rename proc represents a proc that renames fields.
	Rename struct {
		Kind string       `json:"kind" unpack:""`
		Args []Assignment `json:"args"`
	}

	// A Fuse proc represents a proc that turns a zng stream into a dataframe.
	Fuse struct {
		Kind string `json:"kind" unpack:""`
	}

	// A Join proc represents a proc that joins two zng streams.
	Join struct {
		Kind     string       `json:"kind" unpack:""`
		Style    string       `json:"style"`
		LeftKey  Expr         `json:"left_key"`
		RightKey Expr         `json:"right_key"`
		Args     []Assignment `json:"args"`
	}

	// XXX This is a quick and dirty way to get constants into Z.  They are
	// smuggled in as fake procs.  When we refactor this AST into a parser AST
	// proper and a separate kernel DSL, we will clean this up.
	Const struct {
		Kind string `json:"kind" unpack:""`
		Name string `json:"name"`
		Expr Expr   `json:"expr"`
	}

	TypeProc struct {
		Kind string `json:"kind" unpack:""`
		Name string `json:"name"`
		Type Type   `json:"type"`
	}
	// A SQLExpr can be a proc, an expression inside of a SQL FROM clause,
	// or an expression used as a Z value generator.  Currenly, the "select"
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
)

type Method struct {
	Name string `json:"name"`
	Args []Expr `json:"args"`
}

type Case struct {
	Expr Expr `json:"expr"`
	Proc Proc `json:"proc"`
}

type SQLFrom struct {
	Table Expr `json:"table"`
	Alias Expr `json:"alias"`
}

type SQLOrderBy struct {
	Kind  string `json:"kind" unpack:""`
	Keys  []Expr `json:"keys"`
	Order string `json:"order"`
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

func (*Sequential) ProcNode() {}
func (*Parallel) ProcNode()   {}
func (*Switch) ProcNode()     {}
func (*Sort) ProcNode()       {}
func (*Cut) ProcNode()        {}
func (*Pick) ProcNode()       {}
func (*Drop) ProcNode()       {}
func (*Head) ProcNode()       {}
func (*Tail) ProcNode()       {}
func (*Pass) ProcNode()       {}
func (*Filter) ProcNode()     {}
func (*Uniq) ProcNode()       {}
func (*Summarize) ProcNode()  {}
func (*Top) ProcNode()        {}
func (*Put) ProcNode()        {}
func (*Rename) ProcNode()     {}
func (*Fuse) ProcNode()       {}
func (*Join) ProcNode()       {}
func (*Const) ProcNode()      {}
func (*TypeProc) ProcNode()   {}
func (*Call) ProcNode()       {}
func (*Shape) ProcNode()      {}

func (*SQLExpr) ProcNode() {}

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

func DotExprToFieldPath(e Expr) *Path {
	switch e := e.(type) {
	case *BinaryExpr:
		if e.Op == "." {
			lhs := DotExprToFieldPath(e.LHS)
			if lhs == nil {
				return nil
			}
			id, ok := e.RHS.(*Id)
			if !ok {
				return nil
			}
			lhs.Name = append(lhs.Name, id.Name)
			return lhs
		}
		if e.Op == "[" {
			lhs := DotExprToFieldPath(e.LHS)
			if lhs == nil {
				return nil
			}
			id, ok := e.RHS.(*Primitive)
			if !ok || id.Type != "string" {
				return nil
			}
			lhs.Name = append(lhs.Name, id.Text)
			return lhs
		}
	case *Id:
		return &Path{Kind: "Path", Name: []string{e.Name}}
	case *Root:
		return &Path{Kind: "Path", Name: []string{}}
	}
	// This includes a null Expr, which can happen if the AST is missing
	// a field or sets it to null.
	return nil
}

func FieldsOf(e Expr) []field.Static {
	switch e := e.(type) {
	default:
		if f := DotExprToFieldPath(e); f != nil {
			return []field.Static{f.Name}
		}
		return nil
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
	lhs := Expr(&Root{Kind: "Root"})
	for _, name := range f {
		rhs := &Id{
			Kind: "Id",
			Name: name,
		}
		lhs = &BinaryExpr{
			Kind: "BinaryExpr",
			Op:   ".",
			LHS:  lhs,
			RHS:  rhs,
		}
	}
	return lhs
}

func NewAggAssignment(kind string, lval field.Static, arg field.Static) Assignment {
	agg := &Agg{Kind: "Agg", Name: kind}
	if arg != nil {
		agg.Expr = NewDotExpr(arg)
	}
	lhs := lval
	if lhs == nil {
		lhs = field.New(kind)
	}
	return Assignment{
		Kind: "Assignment",
		LHS:  NewDotExpr(lhs),
		RHS:  agg,
	}
}

func FanIn(p Proc) int {
	first := p
	if seq, ok := first.(*Sequential); ok {
		first = seq.Procs[0]
	}
	if p, ok := first.(*Parallel); ok {
		return len(p.Procs)
	}
	if _, ok := first.(*Join); ok {
		return 2
	}
	return 1
}

func FilterToProc(e Expr) *Filter {
	return &Filter{
		Kind: "Filter",
		Expr: e,
	}
}

func (p *Path) String() string {
	return field.Static(p.Name).String()
}

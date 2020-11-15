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
	"encoding/json"

	"github.com/brimsec/zq/field"
)

// XXX Get rid of Node.  Issue #1464.
// A Node is shared by all the AST type nodes.  Currently, this contains the
// op field, as every node type is defined by its op name.
type Node struct {
	Op string `json:"op"`
}

// Proc is the interface implemented by all AST processor nodes.
type Proc interface {
	ProcNode()
}

// BooleanExpr is the interface implement by all AST boolean expression nodes.
type BooleanExpr interface {
	booleanExprNode()
}

// Identifier refers to a syntax element analogous to a programming language
// identifier.  It is currently used exclusively as the RHS of a BinaryExpr "."
// expression though it may have future uses (e.g., enum names or externally
// referred to data e.g, maps to do external joins).
type Identifier struct {
	Node
	Name string `json:"name"`
}

// RootRecord refers to the outer record being operated upon.  Field accesses
// typically begin with the LHS of a "." expression set to a RootRecord.
type RootRecord struct {
	Node
}

type Expression interface {
	exprNode()
}

// Literal is a string representation of a literal value where the
// type field indicates the underlying data type (of the set of all supported
// zng data types, derived from the zng type system and not to be confused with
// the native Go types) and value is a string representation of that value that
// must conform to the provided type.
type Literal struct {
	Node
	Type  string `json:"type"`
	Value string `json:"value"`
}

// ----------------------------------------------------------------------------
// Boolean expressions (aka search expressions or filters)

// A boolean expression is represented by a tree consisting of one
// or more of the following concrete expression nodes.
//
type (
	// A search is a "naked" search term
	Search struct {
		Node
		Text  string  `json:"text"`
		Value Literal `json:"value"`
	}

	// A LogicalAnd node represents a logical and of two subexpressions.
	LogicalAnd struct {
		Node
		Left  BooleanExpr `json:"left"`
		Right BooleanExpr `json:"right"`
	}
	// A LogicalOr node represents a logical or of two subexpressions.
	LogicalOr struct {
		Node
		Left  BooleanExpr `json:"left"`
		Right BooleanExpr `json:"right"`
	}
	// A LogicalNot node represents a logical not of a subexpression.
	LogicalNot struct {
		Node
		Expr BooleanExpr `json:"expr"`
	}
	// A MatchAll node represents a filter that matches all records.
	MatchAll struct {
		Node
	}
	// A CompareAny node represents a comparison operator with all of
	// the fields in a record.
	CompareAny struct {
		Node
		Comparator string `json:"comparator"`
		Recursive  bool   `json:"recursive"`
		Value      Literal
	}
	// A CompareField node represents a comparison operator with a specific
	// field in a record.
	CompareField struct {
		Node
		Comparator string     `json:"comparator"`
		Field      Expression `json:"field"`
		Value      Literal    `json:"value"`
	}
)

// booleanEpxrNode() ensures that only boolean expression nodes can be
// assigned to a BooleanExpr.
//
func (*Search) booleanExprNode()           {}
func (*LogicalAnd) booleanExprNode()       {}
func (*LogicalOr) booleanExprNode()        {}
func (*LogicalNot) booleanExprNode()       {}
func (*MatchAll) booleanExprNode()         {}
func (*CompareAny) booleanExprNode()       {}
func (*CompareField) booleanExprNode()     {}
func (*BinaryExpression) booleanExprNode() {}

type UnaryExpression struct {
	Node
	Operator string     `json:"operator"`
	Operand  Expression `json:"operand"`
}

// A BinaryExpression is any expression of the form "operand operator operand"
// including arithmetic (+, -, *, /), logical operators (and, or),
// comparisons (=, !=, <, <=, >, >=), index operatons (on arrays, sets, and records)
// with operator "[" and a dot expression (".") (on records).
type BinaryExpression struct {
	Node
	Operator string     `json:"operator"`
	LHS      Expression `json:"lhs"`
	RHS      Expression `json:"rhs"`
}

type ConditionalExpression struct {
	Node
	Condition Expression `json:"condition"`
	Then      Expression `json:"then"`
	Else      Expression `json:"else"`
}

type FunctionCall struct {
	Node
	Function string       `json:"function"`
	Args     []Expression `json:"args"`
}

type CastExpression struct {
	Node
	Expr Expression `json:"expr"`
	Type string     `json:"type"`
}

func (*UnaryExpression) exprNode()       {}
func (*BinaryExpression) exprNode()      {}
func (*ConditionalExpression) exprNode() {}
func (*FunctionCall) exprNode()          {}
func (*CastExpression) exprNode()        {}
func (*Literal) exprNode()               {}
func (*Identifier) exprNode()            {}
func (*RootRecord) exprNode()            {}
func (*Assignment) exprNode()            {}
func (*Reducer) exprNode()               {}

// ----------------------------------------------------------------------------
// Procs

// A proc is a node in the flowgraph that takes records in, processes them,
// and produces records as output.
//
type (
	// A SequentialProc node represents a set of procs that receive
	// a stream of records from their parent into the first proc
	// and each subsequent proc processes the output records from the
	// previous proc.
	SequentialProc struct {
		Node
		Procs []Proc `json:"procs"`
	}
	// A ParallelProc node represents a set of procs that each get
	// a stream of records from their parent.
	ParallelProc struct {
		Node
		// If non-zero, MergeOrderField contains the field name on
		// which the branches of this parallel proc should be
		// merged in the order indicated by MergeOrderReverse.
		MergeOrderField   field.Static `json:"merge_order_field"`
		MergeOrderReverse bool         `json:"merge_order_reverse"`
		Procs             []Proc       `json:"procs"`
	}
	// A SortProc node represents a proc that sorts records.
	SortProc struct {
		Node
		Fields     []Expression `json:"fields"`
		SortDir    int          `json:"sortdir"`
		NullsFirst bool         `json:"nullsfirst"`
	}
	// A CutProc node represents a proc that removes fields from each
	// input record where each removed field matches one of the named fields
	// sending each such modified record to its output in the order received.
	CutProc struct {
		Node
		Complement bool         `json:"complement"`
		Fields     []Assignment `json:"fields"`
	}
	// A HeadProc node represents a proc that forwards the indicated number
	// of records then terminates.
	HeadProc struct {
		Node
		Count int `json:"count"`
	}
	// A TailProc node represents a proc that reads all its records from its
	// input transmits the final number of records indicated by the count.
	TailProc struct {
		Node
		Count int `json:"count"`
	}
	// A FilterProc node represents a proc that discards all records that do
	// not match the indicfated filter and forwards all that match to its output.
	FilterProc struct {
		Node
		Filter BooleanExpr `json:"filter"`
	}
	// A PassProc node represents a passthrough proc that mirrors
	// incoming Pull()s on its parent and returns the result.
	PassProc struct {
		Node
	}
	// A UniqProc node represents a proc that discards any record that matches
	// the previous record transmitted.  The Cflag causes the output records
	// to contain a new field called count that contains the number of matched
	// records in that set, similar to the unix shell command uniq.
	UniqProc struct {
		Node
		Cflag bool `json:"cflag"`
	}
	// A GroupByProc node represents a proc that consumes all the records
	// in its input, partitions the records into groups based on the values
	// of the fields specified in the keys field (where the first key is the
	// primary grouping key), and applies reducers (if any) to each group. If the
	// Duration field is non-zero, then the groups are further partioned by time
	// into bins of the duration.  In this case, the primary grouping key is ts.
	// The InputSortDir field indicates that input is sorted (with
	// direction indicated by the sign of the field) in the primary
	// grouping key. In this case, the proc outputs the reducer
	// results from each key as they complete so that large inputs
	// are processed and streamed efficiently.
	// The Limit field specifies the number of different groups that can be
	// aggregated over. When absent, the runtime defaults to an
	// appropriate value.
	// If EmitPart is true, the proc will produce decomposed
	// output results, using the reducer.ResultPart()
	// method. Likewise, if ConsumePart is true, the proc will
	// expect decomposed inputs, using the reducer.ResultPart()
	// method. It is an error for either of these flags to be true
	// if any reducer in Reducers is non-decomposable.
	GroupByProc struct {
		Node
		Duration     Duration     `json:"duration,omitempty"`
		InputSortDir int          `json:"input_sort_dir,omitempty"`
		Limit        int          `json:"limit,omitempty"`
		Keys         []Assignment `json:"keys,omitempty"`
		Reducers     []Assignment `json:"reducers,omitempty"`
		ConsumePart  bool         `json:"consume_part,omitempty"`
		EmitPart     bool         `json:"emit_part,omitempty"`
	}
	// TopProc is similar to proc.SortProc with a few key differences:
	// - It only sorts in descending order.
	// - It utilizes a MaxHeap, immediately discarding records that are not in
	// the top N of the sort.
	// - It has a hidden option (FlushEvery) to sort and emit on every batch.
	TopProc struct {
		Node
		Limit  int          `json:"limit,omitempty"`
		Fields []Expression `json:"fields,omitempty"`
		Flush  bool         `json:"flush"`
	}

	PutProc struct {
		Node
		Clauses []Assignment `json:"clauses"`
	}

	// A RenameProc node represents a proc that renames fields.
	RenameProc struct {
		Node
		Fields []Assignment `json:"fields"`
	}

	// A FuseProc node represents a proc that turns a zng stream into a dataframe.
	FuseProc struct {
		Node
	}
)

type Assignment struct {
	LHS Expression `json:"lhs,omitempty"`
	RHS Expression `json:"rhs"`
}

//XXX TBD: chance to nano.Duration
type Duration struct {
	Seconds int `json:"seconds"`
}

type DurationNode struct {
	Type    string `json:"type"`
	Seconds int    `json:"seconds"`
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v DurationNode
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	d.Seconds = v.Seconds
	return nil
}

func (*SequentialProc) ProcNode() {}
func (*ParallelProc) ProcNode()   {}
func (*SortProc) ProcNode()       {}
func (*CutProc) ProcNode()        {}
func (*HeadProc) ProcNode()       {}
func (*TailProc) ProcNode()       {}
func (*PassProc) ProcNode()       {}
func (*FilterProc) ProcNode()     {}
func (*UniqProc) ProcNode()       {}
func (*GroupByProc) ProcNode()    {}
func (*TopProc) ProcNode()        {}
func (*PutProc) ProcNode()        {}
func (*RenameProc) ProcNode()     {}
func (*FuseProc) ProcNode()       {}

// A Reducer is an AST node that represents a reducer function.  The Operator
// field indicates the aggregation method while the Expr field indicates
// an expression applied to the incoming records that is operated upon by them
// reducer function.  If Expr isn't present, then the reducer doesn't act upon
// a function of the record, e.g., count() counts up records without looking
// into them.
type Reducer struct {
	Node
	Operator string     `json:"operator"`
	Expr     Expression `json:"expr,omitempty"`
	Where    Expression `json:"where,omitempty"`
}

func DotExprToField(n Expression) (field.Static, bool) {
	switch n := n.(type) {
	case nil:
		return nil, true
	case *BinaryExpression:
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
	case *Assignment:
		lhs, ok := DotExprToField(n.LHS)
		if !ok {
			return nil, false
		}
		rhs, ok := DotExprToField(n.RHS)
		if !ok {
			return nil, false
		}
		return append(lhs, rhs...), true
	case *Identifier:
		return field.Static{n.Name}, true
	case *RootRecord:
		return nil, true
	}
	return nil, false
}

func NewDotExpr(f field.Static) Expression {
	lhs := Expression(&RootRecord{Node: Node{Op: "RootRecord"}})
	for _, name := range f {
		rhs := &Identifier{
			Node: Node{"Identifier"},
			Name: name,
		}
		lhs = &BinaryExpression{
			Node:     Node{"BinaryExpr"},
			Operator: ".",
			LHS:      lhs,
			RHS:      rhs,
		}
	}
	return lhs
}

func NewReducerAssignment(op string, lval field.Static, arg field.Static) Assignment {
	reducer := &Reducer{Node: Node{"Reducer"}, Operator: op}
	if arg != nil {
		reducer.Expr = NewDotExpr(arg)
	}
	lhs := lval
	if lhs == nil {
		lhs = field.New(op)
	}
	return Assignment{LHS: NewDotExpr(lhs), RHS: reducer}
}

func FanIn(p Proc) int {
	first := p
	if seq, ok := first.(*SequentialProc); ok {
		first = seq.Procs[0]
	}
	if p, ok := first.(*ParallelProc); ok {
		return len(p.Procs)
	}
	return 1
}

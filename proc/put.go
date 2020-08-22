package proc

import (
	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

// Put is a proc that modifies the record stream with computed values.
// Each new value is called a clause and consists of a field name and
// an expression.  Currently, the field name must be a top-level record
// name, i.e., it cannot be a dotted record access or array reference
// (XXX this will change when we add more comprehensive expression support).
// Each put clause either replaces an existing value in the column specified
// or appends a value as a new column.  Appended values appear as new
// columns in the order that the clause appears in the put expression.
type Put struct {
	Base
	clauses []clause
	// vals is a fixed array to avoid re-allocating for every record
	vals   []zng.Value
	rules  map[int]*putRule
	warned map[string]struct{}
}

// A putRule describes how a given record type is modified by describing
// which input columns should be replaced with which clause expression and
// which clauses should be appended.  The types of each clause expression
// is recorded since a new rule must be created if they change.  Such changes
// aren't typically expected but are possible in the expression language.
type putRule struct {
	typ         *zng.TypeRecord
	clauseTypes []clauseType
	// The clause numbers indexed by input column number that should be
	// replaced, where -1 indicates no replacement.  Nil if there are no
	// replacements.
	replace []int
	// The clause numbers that should be appeneded to the output record
	// ordered left to right.  Nil if nothing need be appended.
	append []int
}

type clauseType struct {
	zng.Type
	container bool
}

type clause struct {
	target string
	eval   expr.ExpressionEvaluator
}

func CompilePutProc(c *Context, parent Proc, node *ast.PutProc) (*Put, error) {
	clauses := make([]clause, len(node.Clauses))
	for k, cl := range node.Clauses {
		var err error
		clauses[k].target = cl.Target
		clauses[k].eval, err = expr.CompileExpr(cl.Expr)
		if err != nil {
			return nil, err
		}
	}
	return &Put{
		Base:    Base{Context: c, Parent: parent},
		clauses: clauses,
		vals:    make([]zng.Value, len(node.Clauses)),
		rules:   make(map[int]*putRule),
		warned:  make(map[string]struct{}),
	}, nil
}

func (p *Put) maybeWarn(err error) {
	s := err.Error()
	_, alreadyWarned := p.warned[s]
	if !alreadyWarned {
		p.Warnings <- s
		p.warned[s] = struct{}{}
	}
}

func (p *Put) eval(in *zng.Record) ([]zng.Value, error) {
	vals := p.vals
	for k, cl := range p.clauses {
		var err error
		vals[k], err = cl.eval(in)
		if err != nil {
			return nil, err
		}
	}
	return vals, nil
}

func (p *Put) buildRule(inType *zng.TypeRecord, vals []zng.Value) (*putRule, error) {
	n := len(inType.Columns)
	cols := make([]zng.Column, n, n+len(p.clauses))
	copy(cols, inType.Columns)
	clauseTypes := make([]clauseType, len(p.clauses))
	nreplace := 0
	replace := make([]int, n)
	for k := range replace {
		replace[k] = -1
	}
	var tail []int
	for k, cl := range p.clauses {
		typ := vals[k].Type
		clauseTypes[k].Type = typ
		clauseTypes[k].container = zng.IsContainerType(typ)
		col := zng.Column{Name: cl.target, Type: typ}
		position, hasCol := inType.ColumnOfField(cl.target)
		if hasCol {
			nreplace++
			replace[position] = k
			cols[position] = col
		} else {
			tail = append(tail, k)
			cols = append(cols, col)
		}
	}
	if nreplace == 0 {
		replace = nil
	}
	typ, err := p.TypeContext.LookupTypeRecord(cols)
	if err != nil {
		return nil, err
	}
	return &putRule{
		typ:         typ,
		clauseTypes: clauseTypes,
		replace:     replace,
		append:      tail,
	}, nil
}

func clauseTypesMatch(types []clauseType, vals []zng.Value) bool {
	for k, typ := range types {
		if vals[k].Type != typ.Type {
			return false
		}
	}
	return true
}

func (p *Put) lookupRule(inType *zng.TypeRecord, vals []zng.Value) (*putRule, error) {
	rule := p.rules[inType.ID()]
	if rule != nil && clauseTypesMatch(rule.clauseTypes, vals) {
		return rule, nil
	}
	var err error
	rule, err = p.buildRule(inType, vals)
	p.rules[inType.ID()] = rule
	return rule, err
}

func (p *Put) put(in *zng.Record) *zng.Record {
	vals, err := p.eval(in)
	if err != nil {
		p.maybeWarn(err)
		return in
	}
	rule, err := p.lookupRule(in.Type, vals)
	if err != nil {
		p.maybeWarn(err)
		return in
	}
	// Start the new output value by either copying or replacing the input values.
	var bytes zcode.Bytes
	if rule.replace == nil {
		// All fields are being appended.
		bytes = in.Raw
	} else {
		// We're overwriting one or more fields.  Travese the
		// replacement vector to determine whether each value should
		// be copied from the input or replaced with a clause result.
		iter := in.ZvalIter()
		for _, clause := range rule.replace {
			item, isContainer, err := iter.Next()
			if err != nil {
				panic(err)
			}
			if clause >= 0 {
				item = vals[clause].Bytes
				isContainer = rule.clauseTypes[clause].container
			}
			if isContainer {
				bytes = zcode.AppendContainer(bytes, item)
			} else {
				bytes = zcode.AppendPrimitive(bytes, item)
			}
		}
	}
	// Finish building the output by appending the remaining clauses if any.
	for clause := range rule.append {
		item := vals[clause].Bytes
		if rule.clauseTypes[clause].container {
			bytes = zcode.AppendContainer(bytes, item)
		} else {
			bytes = zcode.AppendPrimitive(bytes, item)
		}
	}
	return zng.NewRecord(rule.typ, bytes)
}

func (p *Put) Pull() (zbuf.Batch, error) {
	batch, err := p.Get()
	if EOS(batch, err) {
		return nil, err
	}
	recs := make([]*zng.Record, 0, batch.Length())
	for k := 0; k < batch.Length(); k++ {
		in := batch.Index(k)
		recs = append(recs, p.put(in))
	}
	batch.Unref()
	return zbuf.NewArray(recs), nil
}

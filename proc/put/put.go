package put

import (
	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/proc"
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
type Proc struct {
	pctx    *proc.Context
	parent  proc.Interface
	clauses []clause
	// vals is a fixed array to avoid re-allocating for every record
	vals   []zng.Value
	rules  map[int]*putRule
	warned map[string]struct{}
}

// A putRule describes how a given record type is modified by describing
// which input columns should be replaced with which clause expression and
// which clauses should be appended.  The type of each clause expression
// is recorded since a new rule must be created if any of the types change.
// Such changes aren't typically expected but are possible in the expression
// language.
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

func New(pctx *proc.Context, parent proc.Interface, node *ast.PutProc) (proc.Interface, error) {
	clauses := make([]clause, len(node.Clauses))
	for k, cl := range node.Clauses {
		var err error
		clauses[k].target = cl.Target
		clauses[k].eval, err = expr.CompileExpr(cl.Expr)
		if err != nil {
			return nil, err
		}
	}
	return &Proc{
		pctx:    pctx,
		parent:  parent,
		clauses: clauses,
		vals:    make([]zng.Value, len(node.Clauses)),
		rules:   make(map[int]*putRule),
		warned:  make(map[string]struct{}),
	}, nil
}

func (p *Proc) maybeWarn(err error) {
	s := err.Error()
	_, alreadyWarned := p.warned[s]
	if !alreadyWarned {
		p.pctx.Warnings <- s
		p.warned[s] = struct{}{}
	}
}

func (p *Proc) eval(in *zng.Record) ([]zng.Value, error) {
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

func (p *Proc) buildRule(inType *zng.TypeRecord, vals []zng.Value) (*putRule, error) {
	n := len(inType.Columns)
	cols := make([]zng.Column, n, n+len(p.clauses))
	copy(cols, inType.Columns)
	clauseTypes := make([]clauseType, len(p.clauses))
	var nreplace int
	replace := make([]int, n)
	for k := range replace {
		replace[k] = -1
	}
	var tail []int
	for k, cl := range p.clauses {
		typ := vals[k].Type
		clauseTypes[k] = clauseType{typ, zng.IsContainerType(typ)}
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
	typ, err := p.pctx.TypeContext.LookupTypeRecord(cols)
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

func (p *Proc) lookupRule(inType *zng.TypeRecord, vals []zng.Value) (*putRule, error) {
	rule := p.rules[inType.ID()]
	if rule != nil && clauseTypesMatch(rule.clauseTypes, vals) {
		return rule, nil
	}
	rule, err := p.buildRule(inType, vals)
	p.rules[inType.ID()] = rule
	return rule, err
}

func (p *Proc) put(in *zng.Record) *zng.Record {
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
		bytes = make([]byte, len(in.Raw))
		copy(bytes, in.Raw)
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
	for _, clause := range rule.append {
		item := vals[clause].Bytes
		if rule.clauseTypes[clause].container {
			bytes = zcode.AppendContainer(bytes, item)
		} else {
			bytes = zcode.AppendPrimitive(bytes, item)
		}
	}
	return zng.NewRecord(rule.typ, bytes)
}

func (p *Proc) Pull() (zbuf.Batch, error) {
	batch, err := p.parent.Pull()
	if proc.EOS(batch, err) {
		return nil, err
	}
	recs := make([]*zng.Record, 0, batch.Length())
	for k := 0; k < batch.Length(); k++ {
		in := batch.Index(k)
		// Keep is necessary because put can return its argument.
		recs = append(recs, p.put(in).Keep())
	}
	batch.Unref()
	return zbuf.Array(recs), nil
}

func (p *Proc) Done() {
	p.parent.Done()
}

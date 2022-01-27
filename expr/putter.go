package expr

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

// Putter is an Applier that modifies the record stream with computed values.
// Each new value is called a clause and consists of a field name and
// an expression. Each put clause either replaces an existing value in
// the column specified or appends a value as a new column.  Appended
// values appear as new columns in the order that the clause appears
// in the put expression.
type Putter struct {
	zctx    *zed.Context
	builder zcode.Builder
	clauses []Assignment
	// vals is a fixed array to avoid re-allocating for every record
	vals   []zed.Value
	rules  map[int]putRule
	warned map[string]struct{}
}

// A putRule describes how a given record type is modified by describing
// which input columns should be replaced with which clause expression and
// which clauses should be appended.  The type of each clause expression
// is recorded since a new rule must be created if any of the types change.
// Such changes aren't typically expected but are possible in the expression
// language.
type putRule struct {
	typ         zed.Type
	clauseTypes []zed.Type
	step        putStep
}

func NewPutter(zctx *zed.Context, clauses []Assignment) (*Putter, error) {
	for i, p := range clauses {
		if p.LHS.IsEmpty() {
			return nil, fmt.Errorf("put: LHS cannot be 'this' (use 'yield' operator)")
		}
		for j, c := range clauses {
			if i == j {
				continue
			}
			if p.LHS.Equal(c.LHS) {
				return nil, fmt.Errorf("put: multiple assignments to %s", p.LHS)
			}
			if c.LHS.HasStrictPrefix(p.LHS) {
				return nil, fmt.Errorf("put: conflicting nested assignments to %s and %s", p.LHS, c.LHS)
			}
		}
	}
	return &Putter{
		zctx:    zctx,
		clauses: clauses,
		vals:    make([]zed.Value, len(clauses)),
		rules:   make(map[int]putRule),
		warned:  make(map[string]struct{}),
	}, nil
}

func (p *Putter) eval(ectx Context, this *zed.Value) []zed.Value {
	vals := p.vals[:0]
	for _, cl := range p.clauses {
		val := *cl.RHS.Eval(ectx, this)
		// XXX See issue #3370
		//if val.IsQuiet() {
		//	continue
		//}
		vals = append(vals, val)
	}
	return vals
}

// A putStep is a recursive data structure encoding a series of steps to be
// carried out to construct an output record from an input record and
// a slice of evaluated clauses.
type putStep struct {
	op        putOp
	index     int
	container bool
	record    []putStep // for op == record
}

func (p *putStep) append(step putStep) {
	p.record = append(p.record, step)
}

type putOp int

const (
	putFromInput  putOp = iota // copy field from input record
	putFromClause              // copy field from put assignment
	putRecord                  // recurse into record below us
)

func (p *putStep) build(in zcode.Bytes, b *zcode.Builder, vals []zed.Value) zcode.Bytes {
	switch p.op {
	case putRecord:
		b.Reset()
		if err := p.buildRecord(in, b, vals); err != nil {
			return nil
		}
		return b.Bytes()
	default:
		// top-level op must be a record
		panic(fmt.Sprintf("put: unexpected step %v", p.op))
	}
}

func (p *putStep) buildRecord(in zcode.Bytes, b *zcode.Builder, vals []zed.Value) error {
	ig := newGetter(in)

	for _, step := range p.record {
		switch step.op {
		case putFromInput:
			bytes, err := ig.nth(step.index)
			if err != nil {
				return err
			}
			b.Append(bytes)
		case putFromClause:
			b.Append(vals[step.index].Bytes)
		case putRecord:
			b.BeginContainer()
			bytes, err := in, error(nil)
			if step.index >= 0 {
				bytes, err = ig.nth(step.index)
				if err != nil {
					return err
				}
			}
			if err := step.buildRecord(bytes, b, vals); err != nil {
				return err
			}
			b.EndContainer()
		}
	}
	return nil
}

// A getter provides random access to values in a zcode container
// using zcode.Iter. It uses a cursor to avoid quadratic re-seeks for
// the common case where values are fetched sequentially.
type getter struct {
	cursor    int
	container zcode.Bytes
	iter      zcode.Iter
}

func newGetter(cont zcode.Bytes) getter {
	return getter{
		cursor:    -1,
		container: cont,
		iter:      cont.Iter(),
	}
}
func (ig *getter) nth(n int) (zcode.Bytes, error) {
	if n < ig.cursor {
		ig.iter = ig.container.Iter()
	}
	for !ig.iter.Done() {
		zv := ig.iter.Next()
		ig.cursor++
		if ig.cursor == n {
			return zv, nil
		}
	}
	return nil, fmt.Errorf("getter.nth: array index %d out of bounds", n)
}

func findOverwriteClause(path field.Path, clauses []Assignment) (int, field.Path, bool) {
	for i, cand := range clauses {
		if path.Equal(cand.LHS) || cand.LHS.HasStrictPrefix(path) {
			return i, cand.LHS, true
		}
	}
	return -1, nil, false
}

func (p *Putter) deriveSteps(inType *zed.TypeRecord, vals []zed.Value) (putStep, zed.Type) {
	return p.deriveRecordSteps(field.NewEmpty(), inType.Columns, vals)
}

func (p *Putter) deriveRecordSteps(parentPath field.Path, inCols []zed.Column, vals []zed.Value) (putStep, *zed.TypeRecord) {
	s := putStep{op: putRecord}
	cols := make([]zed.Column, 0)

	// First look at all input columns to see which should
	// be copied over and which should be overwritten by
	// assignments.
	for i, inCol := range inCols {
		path := append(parentPath, inCol.Name)
		matchIndex, matchPath, found := findOverwriteClause(path, p.clauses)
		switch {
		// input not overwritten by assignment: copy input value.
		case !found:
			s.append(putStep{
				op:        putFromInput,
				container: zed.IsContainerType(inCol.Type),
				index:     i,
			})
			cols = append(cols, inCol)
		// input field overwritten by non-nested assignment: copy assignment value.
		case len(path) == len(matchPath):
			s.append(putStep{
				op:        putFromClause,
				container: zed.IsContainerType(vals[matchIndex].Type),
				index:     matchIndex,
			})
			cols = append(cols, zed.Column{inCol.Name, vals[matchIndex].Type})
		// input record field overwritten by nested assignment: recurse.
		case len(path) < len(matchPath) && zed.IsRecordType(inCol.Type):
			nestedStep, typ := p.deriveRecordSteps(path, zed.TypeRecordOf(inCol.Type).Columns, vals)
			nestedStep.index = i
			s.append(nestedStep)
			cols = append(cols, zed.Column{inCol.Name, typ})
		// input non-record field overwritten by nested assignment(s): recurse.
		case len(path) < len(matchPath) && !zed.IsRecordType(inCol.Type):
			nestedStep, typ := p.deriveRecordSteps(path, []zed.Column{}, vals)
			nestedStep.index = i
			s.append(nestedStep)
			cols = append(cols, zed.Column{inCol.Name, typ})
		default:
			panic("put: internal error computing record steps")
		}
	}

	appendClause := func(cl Assignment) bool {
		if !cl.LHS.HasPrefix(parentPath) {
			return false
		}
		return !hasField(cl.LHS[len(parentPath)], cols)
	}
	// Then, look at put assignments to see if there are any new fields to append.
	for i, cl := range p.clauses {
		if appendClause(cl) {
			switch {
			// Append value at this level
			case len(cl.LHS) == len(parentPath)+1:
				s.append(putStep{
					op:        putFromClause,
					container: zed.IsContainerType(vals[i].Type),
					index:     i,
				})
				cols = append(cols, zed.Column{cl.LHS[len(parentPath)], vals[i].Type})
			// Appended and nest. For example, this would happen with "put b.c=1" applied to a record {"a": 1}.
			case len(cl.LHS) > len(parentPath)+1:
				path := append(parentPath, cl.LHS[len(parentPath)])
				nestedStep, typ := p.deriveRecordSteps(path, []zed.Column{}, vals)
				nestedStep.index = -1
				cols = append(cols, zed.Column{cl.LHS[len(parentPath)], typ})
				s.append(nestedStep)
			}
		}
	}
	typ, err := p.zctx.LookupTypeRecord(cols)
	if err != nil {
		panic(err)
	}
	return s, typ
}

func hasField(name string, cols []zed.Column) bool {
	for _, col := range cols {
		if col.Name == name {
			return true
		}
	}
	return false
}

func sameTypes(types []zed.Type, vals []zed.Value) bool {
	for k, typ := range types {
		if vals[k].Type != typ {
			return false
		}
	}
	return true
}

func (p *Putter) lookupRule(inType *zed.TypeRecord, vals []zed.Value) putRule {
	rule, ok := p.rules[inType.ID()]
	if ok && sameTypes(rule.clauseTypes, vals) {
		return rule
	}
	step, typ := p.deriveSteps(inType, vals)
	var clauseTypes []zed.Type
	for _, val := range vals {
		clauseTypes = append(clauseTypes, val.Type)
	}
	rule = putRule{typ, clauseTypes, step}
	p.rules[inType.ID()] = rule
	return rule
}

func (p *Putter) Eval(ectx Context, this *zed.Value) *zed.Value {
	recType := zed.TypeRecordOf(this.Type)
	if recType == nil {
		if this.IsError() {
			// propagate errors
			return this
		}
		return ectx.CopyValue(*p.zctx.NewErrorf("put: not a record: %s", zson.MustFormatValue(*this)))
	}
	vals := p.eval(ectx, this)
	rule := p.lookupRule(recType, vals)
	bytes := rule.step.build(this.Bytes, &p.builder, vals)
	return zed.NewValue(rule.typ, bytes)
}

func (*Putter) String() string { return "put" }

func (*Putter) Warning() string { return "" }

package put

import (
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/expr"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

// Put is a proc that modifies the record stream with computed values.
// Each new value is called a clause and consists of a field name and
// an expression. Each put clause either replaces an existing value in
// the column specified or appends a value as a new column.  Appended
// values appear as new columns in the order that the clause appears
// in the put expression.
type Proc struct {
	pctx    *proc.Context
	parent  proc.Interface
	builder zcode.Builder
	clauses []expr.Assignment
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
	step        step
}

func New(pctx *proc.Context, parent proc.Interface, clauses []expr.Assignment) (proc.Interface, error) {
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

	return &Proc{
		pctx:    pctx,
		parent:  parent,
		clauses: clauses,
		vals:    make([]zed.Value, len(clauses)),
		rules:   make(map[int]putRule),
		warned:  make(map[string]struct{}),
	}, nil
}

func (p *Proc) eval(ectx expr.Context, this *zed.Value) []zed.Value {
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

// A step is a recursive data structure encoding a series of steps to be
// carried out to construct an output record from an input record and
// a slice of evaluated clauses.
type step struct {
	op        op
	index     int
	container bool
	record    []step // for op == record
}

func (s *step) append(step step) {
	s.record = append(s.record, step)
}

type op int

const (
	fromInput  op = iota // copy field from input record
	fromClause           // copy field from put assignment
	record               // recurse into record below us
)

func (s step) build(in zcode.Bytes, b *zcode.Builder, vals []zed.Value) zcode.Bytes {
	switch s.op {
	case record:
		b.Reset()
		if err := s.buildRecord(in, b, vals); err != nil {
			return nil
		}
		return b.Bytes()
	default:
		// top-level op must be a record
		panic(fmt.Sprintf("put: unexpected step %v", s.op))
	}
}

func (s step) buildRecord(in zcode.Bytes, b *zcode.Builder, vals []zed.Value) error {
	ig := newGetter(in)

	for _, step := range s.record {
		switch step.op {
		case fromInput:
			bytes, err := ig.nth(step.index)
			if err != nil {
				return err
			}
			if step.container {
				b.AppendContainer(bytes)
			} else {
				b.AppendPrimitive(bytes)
			}
		case fromClause:
			if step.container {
				b.AppendContainer(vals[step.index].Bytes)
			} else {
				b.AppendPrimitive(vals[step.index].Bytes)
			}
		case record:
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
		zv, _ := ig.iter.Next()
		ig.cursor++
		if ig.cursor == n {
			return zv, nil
		}
	}
	return nil, fmt.Errorf("getter.nth: array index %d out of bounds", n)
}

func findOverwriteClause(path field.Path, clauses []expr.Assignment) (int, field.Path, bool) {
	for i, cand := range clauses {
		if path.Equal(cand.LHS) || cand.LHS.HasStrictPrefix(path) {
			return i, cand.LHS, true
		}
	}
	return -1, nil, false
}

func (p *Proc) deriveSteps(inType *zed.TypeRecord, vals []zed.Value) (step, zed.Type) {
	return p.deriveRecordSteps(field.NewEmpty(), inType.Columns, vals)
}

func (p *Proc) deriveRecordSteps(parentPath field.Path, inCols []zed.Column, vals []zed.Value) (step, *zed.TypeRecord) {
	s := step{op: record}
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
			s.append(step{
				op:        fromInput,
				container: zed.IsContainerType(inCol.Type),
				index:     i,
			})
			cols = append(cols, inCol)
		// input field overwritten by non-nested assignment: copy assignment value.
		case len(path) == len(matchPath):
			s.append(step{
				op:        fromClause,
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

	appendClause := func(cl expr.Assignment) bool {
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
				s.append(step{
					op:        fromClause,
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

	typ, err := p.pctx.Zctx.LookupTypeRecord(cols)
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

func (p *Proc) lookupRule(inType *zed.TypeRecord, vals []zed.Value) putRule {
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

func (p *Proc) put(ectx expr.Context, this *zed.Value) *zed.Value {
	recType := zed.TypeRecordOf(this.Type)
	if recType == nil {
		if this.IsError() {
			// propagate errors
			return this
		}
		return ectx.CopyValue(*zed.NewErrorf("put: not a record: %s", zson.MustFormatValue(*this)))
	}
	vals := p.eval(ectx, this)
	rule := p.lookupRule(recType, vals)
	bytes := rule.step.build(this.Bytes, &p.builder, vals)
	return zed.NewValue(rule.typ, bytes)
}

func (p *Proc) Pull() (zbuf.Batch, error) {
	batch, err := p.parent.Pull()
	if batch == nil || err != nil {
		return nil, err
	}
	ectx := batch.Context()
	vals := batch.Values()
	recs := make([]zed.Value, 0, len(vals))
	for i := range vals {
		rec := p.put(ectx, &vals[i])
		if rec.IsQuiet() {
			continue
		}
		// Copy is necessary because put can return its argument.
		recs = append(recs, *rec.Copy())
	}
	batch.Unref()
	return zbuf.NewArray(recs), nil
}

func (p *Proc) Done() {
	p.parent.Done()
}

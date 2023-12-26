package expr

import (
	"encoding/binary"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/expr/dynfield"
	"github.com/brimdata/zed/runtime/expr/pathbuilder"
	"github.com/brimdata/zed/zcode"
)

// Putter is an Evaluator that modifies the record stream with computed values.
// Each new value is called a clause and consists of a field name and
// an expression. Each put clause either replaces an existing value in
// the field specified or appends a value as a new field.  Appended
// values appear as new fields in the order that the clause appears
// in the put expression.
type Putter struct {
	zctx    *zed.Context
	builder zcode.Builder
	clauses []Assignment
	rules   map[int]map[string]pathbuilder.Step
	// vals is a slice to avoid re-allocating for every value
	vals []zed.Value
	// paths is a slice to avoid re-allocating for every path
	paths   dynfield.List
	scratch []byte
}

func NewPutter(zctx *zed.Context, clauses []Assignment) *Putter {
	return &Putter{
		zctx:    zctx,
		clauses: clauses,
		vals:    make([]zed.Value, len(clauses)),
		rules:   make(map[int]map[string]pathbuilder.Step),
	}
}

func (p *Putter) Eval(ectx Context, this *zed.Value) *zed.Value {
	if !this.IsContainer() {
		if this.IsError() {
			// propagate errors
			return this
		}
		return ectx.CopyValue(*p.zctx.WrapError("put: not a puttable element", this))
	}
	paths, vals, err := p.eval(ectx, this)
	if err != nil {
		return ectx.CopyValue(*p.zctx.WrapError(fmt.Sprintf("put: %s", err), this))
	}
	if len(vals) == 0 {
		return this
	}
	step, err := p.lookupRule(this.Type, paths, vals)
	if err != nil {
		return ectx.CopyValue(*p.zctx.WrapError(err.Error(), this))
	}
	p.builder.Reset()
	typ, err := step.Build(p.zctx, &p.builder, this.Bytes(), vals)
	if err != nil {
		return ectx.CopyValue(*p.zctx.WrapError(err.Error(), this))
	}
	return ectx.NewValue(typ, p.builder.Bytes())
}

func (p *Putter) eval(ectx Context, this *zed.Value) (dynfield.List, []zed.Value, error) {
	p.vals = p.vals[:0]
	p.paths = p.paths[:0]
	for _, cl := range p.clauses {
		val := *cl.RHS.Eval(ectx, this)
		if val.IsQuiet() {
			continue
		}
		p.vals = append(p.vals, val)
		path, err := cl.LHS.Eval(ectx, this)
		if err != nil {
			return nil, nil, err
		}
		p.paths = append(p.paths, path)
	}
	return p.paths, p.vals, nil
}

func (p *Putter) lookupRule(inType zed.Type, fields dynfield.List, vals []zed.Value) (pathbuilder.Step, error) {
	m, ok := p.rules[inType.ID()]
	if !ok {
		m = make(map[string]pathbuilder.Step)
		p.rules[inType.ID()] = m
	}
	p.scratch = encodePaths(p.scratch[:0], fields, vals)
	if rule, ok := m[string(p.scratch)]; ok {
		return rule, nil
	}
	step, err := pathbuilder.New(inType, fields, vals)
	if err != nil {
		return nil, err
	}
	p.rules[inType.ID()][string(p.scratch)] = step
	return step, nil
}

func encodePaths(b []byte, fields dynfield.List, vals []zed.Value) []byte {
	for i := range fields {
		if i > 0 {
			b = append(b, ',')
		}
		b = fields[i].Append(b)
		b = append(b, ':')
		b = binary.AppendVarint(b, int64(vals[i].Type.ID()))
	}
	return b
}

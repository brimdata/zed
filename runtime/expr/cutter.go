package expr

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
)

type Cutter struct {
	zctx        *zed.Context
	fieldRefs   field.List
	fieldExprs  []Evaluator
	lvals       []*Lval
	outTypes    *zed.TypeVectorTable
	recordTypes map[int]*zed.TypeRecord
	typeCache   []zed.Type

	builders     map[string]*zed.RecordBuilder
	droppers     map[string]*Dropper
	dropperCache []*Dropper
	dirty        bool
	quiet        bool
}

// NewCutter returns a Cutter for fieldnames. If complement is true,
// the Cutter copies fields that are not in fieldnames. If complement
// is false, the Cutter copies any fields in fieldnames, where targets
// specifies the copied field names.
func NewCutter(zctx *zed.Context, fieldRefs []*Lval, fieldExprs []Evaluator) *Cutter {
	n := len(fieldRefs)
	return &Cutter{
		zctx:         zctx,
		builders:     make(map[string]*zed.RecordBuilder),
		fieldRefs:    make(field.List, n),
		fieldExprs:   fieldExprs,
		lvals:        fieldRefs,
		outTypes:     zed.NewTypeVectorTable(),
		recordTypes:  make(map[int]*zed.TypeRecord),
		typeCache:    make([]zed.Type, n),
		droppers:     make(map[string]*Dropper),
		dropperCache: make([]*Dropper, n),
	}
}

func (c *Cutter) Quiet() {
	c.quiet = true
}

func (c *Cutter) FoundCut() bool {
	return c.dirty
}

// Apply returns a new record comprising fields copied from in according to the
// receiver's configuration.  If the resulting record would be empty, Apply
// returns zed.Missing.
func (c *Cutter) Eval(ectx Context, in *zed.Value) *zed.Value {
	rb, paths, err := c.lookupBuilder(ectx, in)
	if err != nil {
		return ectx.CopyValue(*c.zctx.WrapError(fmt.Sprintf("cut: %s", err), in))
	}
	types := c.typeCache
	rb.Reset()
	droppers := c.dropperCache[:0]
	for k, e := range c.fieldExprs {
		val := e.Eval(ectx, in)
		if val.IsQuiet() {
			// ignore this field
			pathID := paths[k].String()
			if c.droppers[pathID] == nil {
				c.droppers[pathID] = NewDropper(c.zctx, field.List{paths[k]})
			}
			droppers = append(droppers, c.droppers[pathID])
			rb.Append(val.Bytes())
			types[k] = zed.TypeNull
			continue
		}
		rb.Append(val.Bytes())
		types[k] = val.Type
	}
	// check paths
	bytes, err := rb.Encode()
	if err != nil {
		panic(err)
	}
	typ := c.lookupTypeRecord(types, rb)
	rec := ectx.NewValue(typ, bytes)
	for _, d := range droppers {
		rec = d.Eval(ectx, rec)
	}
	if !rec.IsError() {
		c.dirty = true
	}
	return rec
}

func (c *Cutter) lookupBuilder(ectx Context, in *zed.Value) (*zed.RecordBuilder, field.List, error) {
	paths := c.fieldRefs[:0]
	for _, p := range c.lvals {
		path, err := p.Eval(ectx, in)
		if err != nil {
			return nil, nil, err
		}
		if path.IsEmpty() {
			return nil, nil, errors.New("'this' not allowed (use record literal)")
		}
		paths = append(paths, path)
	}
	builder, ok := c.builders[paths.String()]
	if !ok {
		var err error
		if builder, err = zed.NewRecordBuilder(c.zctx, paths); err != nil {
			return nil, nil, err
		}
		c.builders[paths.String()] = builder
	}
	return builder, paths, nil
}

func (c *Cutter) lookupTypeRecord(types []zed.Type, builder *zed.RecordBuilder) *zed.TypeRecord {
	id := c.outTypes.Lookup(types)
	typ, ok := c.recordTypes[id]
	if !ok {
		typ = builder.Type(types)
		c.recordTypes[id] = typ
	}
	return typ
}

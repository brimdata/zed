package expr

import (
	"errors"
	"fmt"
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
)

type Cutter struct {
	zctx        *zed.Context
	builder     *zed.ColumnBuilder
	fieldRefs   field.List
	fieldExprs  []Evaluator
	typeCache   []zed.Type
	outTypes    *zed.TypeVectorTable
	recordTypes map[int]*zed.TypeRecord

	droppers     []*Dropper
	dropperCache []*Dropper
	dirty        bool
	quiet        bool
}

// NewCutter returns a Cutter for fieldnames. If complement is true,
// the Cutter copies fields that are not in fieldnames. If complement
// is false, the Cutter copies any fields in fieldnames, where targets
// specifies the copied field names.
func NewCutter(zctx *zed.Context, fieldRefs field.List, fieldExprs []Evaluator) (*Cutter, error) {
	for _, f := range fieldRefs {
		if f.IsEmpty() {
			return nil, errors.New("cut: 'this' not allowed (use record literal)")
		}
	}
	var b *zed.ColumnBuilder
	if len(fieldRefs) == 0 || !fieldRefs[0].IsEmpty() {
		// A root field will cause NewColumnBuilder to panic.
		var err error
		b, err = zed.NewColumnBuilder(zctx, fieldRefs)
		if err != nil {
			return nil, err
		}
	}
	n := len(fieldRefs)
	return &Cutter{
		zctx:         zctx,
		builder:      b,
		fieldRefs:    fieldRefs,
		fieldExprs:   fieldExprs,
		typeCache:    make([]zed.Type, len(fieldRefs)),
		outTypes:     zed.NewTypeVectorTable(),
		recordTypes:  make(map[int]*zed.TypeRecord),
		droppers:     make([]*Dropper, n),
		dropperCache: make([]*Dropper, n),
	}, nil
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
	types := c.typeCache
	b := c.builder
	b.Reset()
	droppers := c.dropperCache[:0]
	for k, e := range c.fieldExprs {
		val := e.Eval(ectx, in)
		if val.IsQuiet() {
			// ignore this field
			if c.droppers[k] == nil {
				c.droppers[k] = NewDropper(c.zctx, c.fieldRefs[k:k+1])
			}
			droppers = append(droppers, c.droppers[k])
			b.Append(val.Bytes)
			types[k] = zed.TypeNull
			continue
		}
		b.Append(val.Bytes)
		types[k] = val.Type
	}
	bytes, err := b.Encode()
	if err != nil {
		panic(err)
	}
	rec := zed.NewValue(c.lookupTypeRecord(types), bytes)
	for _, d := range droppers {
		rec = d.Eval(ectx, rec)
	}
	if !rec.IsError() {
		c.dirty = true
	}
	return rec
}

func (c *Cutter) lookupTypeRecord(types []zed.Type) *zed.TypeRecord {
	id := c.outTypes.Lookup(types)
	typ, ok := c.recordTypes[id]
	if !ok {
		cols := c.builder.TypedColumns(types)
		typ = c.zctx.MustLookupTypeRecord(cols)
		c.recordTypes[id] = typ
	}
	return typ
}

func fieldList(fields []Evaluator) string {
	var each []string
	for _, fieldExpr := range fields {
		s := "<not a field>"
		if f, err := DotExprToField(fieldExpr); err == nil {
			s = f.String()
		}
		each = append(each, s)
	}
	return strings.Join(each, ",")
}

func (_ *Cutter) String() string { return "cut" }

func (c *Cutter) Warning() string {
	if c.quiet || c.FoundCut() {
		return ""
	}
	return fmt.Sprintf("no record found with columns %s", fieldList(c.fieldExprs))
}

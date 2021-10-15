package expr

import (
	"errors"
	"fmt"
	"strings"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/field"
	"github.com/brimdata/zed/zcode"
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
	if len(fieldRefs) > 1 {
		for _, f := range fieldRefs {
			if f.IsRoot() {
				return nil, errors.New("cannot assign to . when cutting multiple values")
			}
		}
	}
	var b *zed.ColumnBuilder
	if len(fieldRefs) == 0 || !fieldRefs[0].IsRoot() {
		// A root field will cause NewColumnBuilder to panic.
		var err error
		b, err = zed.NewColumnBuilder(zctx, fieldRefs)
		if err != nil {
			return nil, err
		}
	}
	return &Cutter{
		zctx:        zctx,
		builder:     b,
		fieldRefs:   fieldRefs,
		fieldExprs:  fieldExprs,
		typeCache:   make([]zed.Type, len(fieldRefs)),
		outTypes:    zed.NewTypeVectorTable(),
		recordTypes: make(map[int]*zed.TypeRecord),
	}, nil
}

func (c *Cutter) AllowPartialCuts() {
	n := len(c.fieldRefs)
	c.droppers = make([]*Dropper, n)
	c.dropperCache = make([]*Dropper, n)
}

func (c *Cutter) Quiet() {
	c.quiet = true
}

func (c *Cutter) FoundCut() bool {
	return c.dirty
}

// Apply returns a new record comprising fields copied from in according to the
// receiver's configuration.  If the resulting record would be empty, Apply
// returns nil.
func (c *Cutter) Apply(in *zed.Record) (*zed.Record, error) {
	if len(c.fieldRefs) == 1 && c.fieldRefs[0].IsRoot() {
		zv, err := c.fieldExprs[0].Eval(in)
		if err != nil {
			if err == zed.ErrMissing {
				return nil, nil
			}
			return nil, err
		}
		recType, ok := zed.AliasOf(zv.Type).(*zed.TypeRecord)
		if !ok {
			return nil, errors.New("cannot cut a non-record to .")
		}
		if zv.IsUnset() {
			return nil, errors.New("cannot cut an unset value to .")
		}
		c.dirty = true
		return zed.NewRecord(recType, append(zcode.Bytes{}, zv.Bytes...)), nil
	}
	types := c.typeCache
	b := c.builder
	b.Reset()
	droppers := c.dropperCache[:0]
	for k, e := range c.fieldExprs {
		zv, err := e.Eval(in)
		if err != nil {
			if err == zed.ErrMissing {
				if c.droppers != nil {
					if c.droppers[k] == nil {
						c.droppers[k] = NewDropper(c.zctx, c.fieldRefs[k:k+1])
					}
					droppers = append(droppers, c.droppers[k])
					// ignore this record
					b.Append(zv.Bytes, false)
					types[k] = zed.TypeNull
					continue
				}
				err = nil
			}
			return nil, err
		}
		b.Append(zv.Bytes, zv.IsContainer())
		types[k] = zv.Type
	}
	typ, err := c.lookupTypeRecord(types)
	if err != nil {
		return nil, err
	}
	zv, err := b.Encode()
	if err != nil {
		return nil, err
	}
	rec := zed.NewRecord(typ, zv)
	for _, d := range droppers {
		r, err := d.Apply(rec)
		if err != nil {
			return nil, err
		}
		rec = r
	}
	if rec != nil {
		c.dirty = true
	}
	return rec, nil
}

func (c *Cutter) lookupTypeRecord(types []zed.Type) (*zed.TypeRecord, error) {
	id := c.outTypes.Lookup(types)
	typ, ok := c.recordTypes[id]
	if !ok {
		cols := c.builder.TypedColumns(types)
		var err error
		typ, err = c.zctx.LookupTypeRecord(cols)
		if err != nil {
			return nil, err
		}
		c.recordTypes[id] = typ
	}
	return typ, nil
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

func (c *Cutter) Eval(rec *zed.Record) (zed.Value, error) {
	out, err := c.Apply(rec)
	if err != nil {
		return zed.Value{}, err
	}
	if out == nil {
		return zed.Value{}, zed.ErrMissing
	}
	return *out, nil
}

package vng

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
)

type Metadata interface {
	Type(*zed.Context) zed.Type
	Len() uint32
}

type Record struct {
	Length uint32
	Fields []Field
}

func (r *Record) Type(zctx *zed.Context) zed.Type {
	fields := make([]zed.Field, 0, len(r.Fields))
	for _, field := range r.Fields {
		typ := field.Values.Type(zctx)
		fields = append(fields, zed.Field{Name: field.Name, Type: typ})
	}
	return zctx.MustLookupTypeRecord(fields)
}

func (r *Record) Len() uint32 {
	return r.Length
}

func (r *Record) LookupField(name string) *Field {
	for k, field := range r.Fields {
		if field.Name == name {
			return &r.Fields[k]
		}
	}
	return nil
}

func (r *Record) Lookup(path field.Path) *Field {
	var f *Field
	for _, name := range path {
		f = r.LookupField(name)
		if f == nil {
			return nil
		}
		if next, ok := Under(f.Values).(*Record); ok {
			r = next
		} else {
			break
		}
	}
	return f
}

func Under(meta Metadata) Metadata {
	for {
		switch inner := meta.(type) {
		case *Named:
			meta = inner.Values
		case *Nulls:
			meta = inner.Values
		default:
			return meta
		}
	}
}

type Field struct {
	Name   string
	Values Metadata
}

type Array struct {
	Length  uint32
	Lengths Segment
	Values  Metadata
}

func (a *Array) Type(zctx *zed.Context) zed.Type {
	return zctx.LookupTypeArray(a.Values.Type(zctx))
}

func (a *Array) Len() uint32 {
	return a.Length
}

type Set Array

func (s *Set) Type(zctx *zed.Context) zed.Type {
	return zctx.LookupTypeSet(s.Values.Type(zctx))
}

func (s *Set) Len() uint32 {
	return s.Length
}

type Map struct {
	Length  uint32
	Lengths Segment
	Keys    Metadata
	Values  Metadata
}

func (m *Map) Type(zctx *zed.Context) zed.Type {
	keyType := m.Keys.Type(zctx)
	valType := m.Values.Type(zctx)
	return zctx.LookupTypeMap(keyType, valType)
}

func (m *Map) Len() uint32 {
	return m.Length
}

type Union struct {
	Length uint32
	Tags   Segment
	Values []Metadata
}

func (u *Union) Type(zctx *zed.Context) zed.Type {
	types := make([]zed.Type, 0, len(u.Values))
	for _, value := range u.Values {
		types = append(types, value.Type(zctx))
	}
	return zctx.LookupTypeUnion(types)
}

func (u *Union) Len() uint32 {
	return u.Length
}

type Named struct {
	Name   string
	Values Metadata
}

func (n *Named) Type(zctx *zed.Context) zed.Type {
	t, err := zctx.LookupTypeNamed(n.Name, n.Values.Type(zctx))
	if err != nil {
		panic(err) //XXX
	}
	return t
}

func (n *Named) Len() uint32 {
	return n.Values.Len()
}

type Error struct {
	Values Metadata
}

func (e *Error) Type(zctx *zed.Context) zed.Type {
	return zctx.LookupTypeError(e.Values.Type(zctx))
}

func (e *Error) Len() uint32 {
	return e.Values.Len()
}

type DictEntry struct {
	Value zed.Value
	Count uint32
}

type Primitive struct {
	Typ      zed.Type `zed:"Type"`
	Location Segment
	Dict     []DictEntry
	Min      *zed.Value
	Max      *zed.Value
	Count    uint32
}

func (p *Primitive) Type(zctx *zed.Context) zed.Type {
	return p.Typ
}

func (p *Primitive) Len() uint32 {
	return p.Count
}

type Nulls struct {
	Runs   Segment
	Values Metadata
	Count  uint32 // Count of nulls
}

func (n *Nulls) Type(zctx *zed.Context) zed.Type {
	return n.Values.Type(zctx)
}

func (n *Nulls) Len() uint32 {
	return n.Count + n.Values.Len()
}

type Const struct {
	Value zed.Value
	Count uint32
}

func (c *Const) Type(zctx *zed.Context) zed.Type {
	return c.Value.Type()
}

func (c *Const) Len() uint32 {
	return c.Count
}

type Variant struct {
	Tags   Segment
	Values []Metadata
	Length uint32
}

var _ Metadata = (*Variant)(nil)

func (*Variant) Type(zctx *zed.Context) zed.Type {
	panic("Type should not be called on Variant")
}

func (v *Variant) Len() uint32 {
	return v.Length
}

var Template = []interface{}{
	Record{},
	Array{},
	Set{},
	Map{},
	Union{},
	Primitive{},
	Named{},
	Error{},
	Nulls{},
	Const{},
	Variant{},
}

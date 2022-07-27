package vector

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
)

type Metadata interface {
	Type(*zed.Context) zed.Type
}

type Record struct {
	Fields []Field
}

func (r *Record) Type(zctx *zed.Context) zed.Type {
	cols := make([]zed.Column, 0, len(r.Fields))
	for _, field := range r.Fields {
		typ := field.Values.Type(zctx)
		cols = append(cols, zed.Column{Name: field.Name, Type: typ})
	}
	return zctx.MustLookupTypeRecord(cols)
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
		named, ok := meta.(*Named)
		if !ok {
			return meta
		}
		meta = named.Values
	}
}

type Field struct {
	Presence []Segment
	Name     string
	Values   Metadata
	Empty    bool
}

type Array struct {
	Lengths []Segment
	Values  Metadata
}

func (a *Array) Type(zctx *zed.Context) zed.Type {
	return zctx.LookupTypeArray(a.Values.Type(zctx))
}

type Set Array

func (s *Set) Type(zctx *zed.Context) zed.Type {
	return zctx.LookupTypeSet(s.Values.Type(zctx))
}

type Union struct {
	Presence []Segment
	Tags     []Segment
	Values   []Metadata
}

func (u *Union) Type(zctx *zed.Context) zed.Type {
	types := make([]zed.Type, 0, len(u.Values))
	for _, value := range u.Values {
		types = append(types, value.Type(zctx))
	}
	return zctx.LookupTypeUnion(types)
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

type Primitive struct {
	Typ    zed.Type `zed:"Type"`
	Segmap []Segment
}

func (p *Primitive) Type(zctx *zed.Context) zed.Type {
	return p.Typ
}

var Template = []interface{}{
	Record{},
	Array{},
	Set{},
	Union{},
	Primitive{},
	Named{},
}

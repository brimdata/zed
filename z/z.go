package z

import (
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type Context struct {
	*resolver.Context
}

type Field struct {
	zng.Column
	zcode.Bytes
	fields []Field
}

func NewContext() Context {
	return Context{resolver.NewContext()}
}

func Int64(key string, val int64) Field {
	return Field{zng.Column{key, zng.TypeInt64}, zng.EncodeInt(val), nil}
}

func String(key string, val string) Field {
	return Field{zng.Column{key, zng.TypeString}, zng.EncodeString(val), nil}
}

func (c *Context) zctx() *resolver.Context {
	if c.Context == nil {
		c.Context = resolver.NewContext()
	}
	return c.Context
}

func (c *Context) recType(fields ...Field) *zng.TypeRecord {
	cols := make([]zng.Column, len(fields))
	for k := range fields {
		cols[k] = fields[k].Column
	}
	typ, err := c.zctx().LookupTypeRecord(cols)
	if err != nil {
		panic(err.Error())
	}
	return typ
}

func (c *Context) RecordField(key string, fields ...Field) Field {
	return Field{zng.Column{key, c.recType(fields...)}, nil, fields}
}

func (c *Context) Record(fields ...Field) *zng.Record {
	typ := c.recType(fields...)
	var b zcode.Builder
	c.build(&b, fields...)
	body, _ := b.Bytes().ContainerBody()
	return zng.NewRecord(typ, body)
}

func (c *Context) build(b *zcode.Builder, fields ...Field) {
	b.BeginContainer()
	for _, f := range fields {
		if f.Bytes != nil {
			b.AppendPrimitive(f.Bytes)
			continue
		}
		switch typ := f.Column.Type.(type) {
		case *zng.TypeRecord:
			c.build(b, f.fields...)
		default:
			panic("not implemented: " + typ.String())
		}

	}
	b.EndContainer()
}

package z

import (
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type Container struct {
	*resolver.Context
}

type Field struct {
	zng.Column
	zcode.Bytes
	fields []Field
}

func Int64(key string, val int64) Field {
	return Field{zng.Column{key, zng.TypeInt64}, zng.EncodeInt(val), nil}
}

func Int64v(val int64) Field {
	return Int64("", val)
}
func String(key string, val string) Field {
	return Field{zng.Column{key, zng.TypeString}, zng.EncodeString(val), nil}
}

func Stringv(val string) Field {
	return String("", val)
}

func (c *Container) zctx() *resolver.Context {
	if c.Context == nil {
		c.Context = resolver.NewContext()
	}
	return c.Context
}

func (c *Container) recType(fields ...Field) *zng.TypeRecord {
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

func (c *Container) arrayType(fields ...Field) *zng.TypeArray {
	// XXX we should check the types of all the fields and
	// convert to a union if the types are mixed
	return c.zctx().LookupTypeArray(fields[0].Column.Type)
}

func (c *Container) Record(key string, fields ...Field) Field {
	return Field{zng.Column{key, c.recType(fields...)}, nil, fields}
}

func (c *Container) Array(key string, fields ...Field) Field {
	return Field{zng.Column{key, c.arrayType(fields...)}, nil, fields}
}

func (c *Container) NewRecord(fields ...Field) *zng.Record {
	typ := c.recType(fields...)
	var b zcode.Builder
	c.build(&b, fields...)
	body, _ := b.Bytes().ContainerBody()
	return zng.NewRecord(typ, body)
}

func (c *Container) build(b *zcode.Builder, fields ...Field) {
	b.BeginContainer()
	for _, f := range fields {
		if f.Bytes != nil {
			b.AppendPrimitive(f.Bytes)
			continue
		}
		switch typ := f.Column.Type.(type) {
		case *zng.TypeRecord:
			c.build(b, f.fields...)
		case *zng.TypeArray:
			c.build(b, f.fields...)
		default:
			panic("not implemented: " + typ.String())
		}

	}
	b.EndContainer()
}

package zfmt

import (
	astzed "github.com/brimdata/zed/compiler/ast/zed"
	"github.com/brimdata/zed/zson"
)

type canonZed struct {
	formatter
}

// XXX this needs to change when we use the zson values from the ast
func (c *canonZed) literal(e astzed.Primitive) {
	switch e.Type {
	case "string", "error":
		c.write("\"%s\"", e.Text)
	case "regexp":
		c.write("/%s/", e.Text)
	default:
		//XXX need decorators for non-implied
		c.write("%s", e.Text)

	}
}

func (c *canonZed) fieldpath(path []string) {
	if len(path) == 0 {
		c.write("this")
		return
	}
	for k, s := range path {
		if zson.IsIdentifier(s) {
			if k != 0 {
				c.write(".")
			}
			c.write(s)
		} else {
			if k == 0 {
				c.write(".")
			}
			c.write("[%q]", s)
		}
	}
}

func (c *canonZed) typ(t astzed.Type) {
	switch t := t.(type) {
	case *astzed.TypePrimitive:
		c.write(t.Name)
	case *astzed.TypeRecord:
		c.write("{")
		c.typeFields(t.Fields)
		c.write("}")
	case *astzed.TypeArray:
		c.write("[")
		c.typ(t.Type)
		c.write("]")
	case *astzed.TypeSet:
		c.write("|[")
		c.typ(t.Type)
		c.write("]|")
	case *astzed.TypeUnion:
		c.write("(")
		c.types(t.Types)
		c.write(")")
	case *astzed.TypeEnum:
		//XXX need to figure out Zed syntax for enum literal which may
		// be different than zson, requiring some ast adjustments.
		c.write("TBD:ENUM")
	case *astzed.TypeMap:
		c.write("|{")
		c.typ(t.KeyType)
		c.write(":")
		c.typ(t.ValType)
		c.write("}|")
	case *astzed.TypeNull:
		c.write("null")
	case *astzed.TypeDef:
		c.write("%s=(", t.Name)
		c.typ(t.Type)
		c.write(")")
	case *astzed.TypeName:
		c.write(t.Name)
	}
}

func (c *canonZed) typeFields(fields []astzed.TypeField) {
	for k, f := range fields {
		if k != 0 {
			c.write(",")
		}
		c.write("%s:", zson.QuotedName(f.Name))
		c.typ(f.Type)
	}
}

func (c *canonZed) types(types []astzed.Type) {
	for k, t := range types {
		if k != 0 {
			c.write(",")
		}
		c.typ(t)
	}
}

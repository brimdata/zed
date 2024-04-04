package parser

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/brimdata/zed/compiler/ast"
	"github.com/brimdata/zed/compiler/ast/node"
	astzed "github.com/brimdata/zed/compiler/ast/zed"
)

func newBase(c *current) node.Base {
	return node.Base{
		Start: c.pos.offset,
		End:   c.pos.offset + len(c.text),
	}
}

func sliceOf[E any](s any) []E {
	arr := s.([]any)
	out := make([]E, len(arr))
	for i, el := range arr {
		out[i] = el.(E)
	}
	return out
}

func newPrimitive(c *current, typ, text string) *astzed.Primitive {
	return &astzed.Primitive{
		Base: newBase(c),
		Kind: "Primitive",
		Type: typ,
		Text: text,
	}
}

func makeBinaryExprChain(c *current, first, rest any) any {
	ret := first.(ast.Expr)
	for _, p := range rest.([]any) {
		part := p.([]any)
		// XXX this current thing isn't right.
		ret = newBinaryExpr(c, part[0].(string), ret, part[1].(ast.Expr))
	}
	return ret
}

func newBinaryExpr(c *current, op, lhs, rhs any) *ast.BinaryExpr {
	return &ast.BinaryExpr{
		Base: newBase(c),
		Kind: "BinaryExpr",
		Op:   op.(string),
		LHS:  lhs.(ast.Expr),
		RHS:  rhs.(ast.Expr),
	}
}

func makeArgMap(args interface{}) (interface{}, error) {
	m := make(map[string]interface{})
	for _, a := range args.([]interface{}) {
		arg := a.(map[string]interface{})
		name := arg["name"].(string)
		if _, ok := m[name]; ok {
			return nil, fmt.Errorf("Duplicate argument -%s", name)
		}
		m[name] = arg["value"]
	}
	return m, nil
}

func makeTemplateExprChain(in interface{}) interface{} {
	rest := in.([]interface{})
	ret := rest[0]
	for _, part := range rest[1:] {
		ret = map[string]interface{}{
			"kind": "BinaryExpr",
			"op":   "+",
			"lhs":  ret,
			"rhs":  part,
		}
	}
	return ret
}

func joinChars(in interface{}) string {
	str := bytes.Buffer{}
	for _, i := range in.([]interface{}) {
		// handle joining bytes or strings
		if s, ok := i.([]byte); ok {
			str.Write(s)
		} else {
			str.WriteString(i.(string))
		}
	}
	return str.String()
}

func parseInt(v interface{}) interface{} {
	num := v.(string)
	i, err := strconv.Atoi(num)
	if err != nil {
		return nil
	}

	return i
}

func OR(a, b interface{}) interface{} {
	if a != nil {
		return a
	}

	return b
}

func makeUnicodeChar(chars interface{}) string {
	var r rune
	for _, char := range chars.([]interface{}) {
		if char != nil {
			var v byte
			ch := char.([]byte)[0]
			switch {
			case ch >= '0' && ch <= '9':
				v = ch - '0'
			case ch >= 'a' && ch <= 'f':
				v = ch - 'a' + 10
			case ch >= 'A' && ch <= 'F':
				v = ch - 'A' + 10
			}
			r = (16 * r) + rune(v)
		}
	}

	return string(r)
}

func newAssert(expr any, text string) *ast.Yield {
	// 'assert EXPR' is equivalent to
	// 'yield EXPR ? this : error({message: "assertion failed", "expr": EXPR_text, "on": this}'
	// where EXPR_text is the literal text of EXPR.
	return &ast.Yield{
		Kind: "Yield",
		Exprs: []ast.Expr{
			&ast.Conditional{
				Kind: "Conditional",
				Cond: expr.(ast.Expr),
				Then: &ast.ID{Kind: "ID", Name: "this"},
				Else: &ast.Call{
					Kind: "Call",
					Name: "error",
					Args: []ast.Expr{&ast.RecordExpr{
						Kind: "RecordExpr",
						Elems: []ast.RecordElem{
							&ast.Field{
								Kind:  "Field",
								Name:  "message",
								Value: &astzed.Primitive{Kind: "Primitive", Text: "assertion failed", Type: "string"},
							},
							&ast.Field{
								Kind:  "Field",
								Name:  "expr",
								Value: &astzed.Primitive{Kind: "Primitive", Text: text, Type: "string"},
							},
							&ast.Field{
								Kind:  "Field",
								Name:  "on",
								Value: &ast.ID{Kind: "ID", Name: "this"},
							},
						}},
					},
				},
			},
		},
	}
}

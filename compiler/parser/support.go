package parser

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/brimdata/zed/compiler/ast"
	astzed "github.com/brimdata/zed/compiler/ast/zed"
)

func sliceOf[E any](s any) []E {
	if s == nil {
		return nil
	}
	slice := s.([]any)
	out := make([]E, len(slice))
	for i, el := range slice {
		out[i] = el.(E)
	}
	return out
}

func newPrimitive(c *current, typ, text string) *astzed.Primitive {
	return &astzed.Primitive{
		Kind:    "Primitive",
		Type:    typ,
		Text:    text,
		TextPos: c.pos.offset,
	}
}

func makeBinaryExprChain(first, rest any) any {
	ret := first.(ast.Expr)
	for _, p := range rest.([]any) {
		part := p.([]any)
		ret = &ast.BinaryExpr{
			Kind: "BinaryExpr",
			Op:   part[0].(string),
			LHS:  ret,
			RHS:  part[1].(ast.Expr),
		}
	}
	return ret
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

func makeTemplateExprChain(in any) any {
	rest := in.([]any)
	ret := rest[0].(ast.Expr)
	for _, part := range rest[1:] {
		ret = &ast.BinaryExpr{
			Kind: "BinaryExpr",
			Op:   "+",
			LHS:  ret,
			RHS:  part.(ast.Expr),
		}
	}
	return ret
}

func newCall(c *current, name, args, where any) ast.Expr {
	call := &ast.Call{
		Kind:    "Call",
		Name:    name.(string),
		NamePos: c.pos.offset,
		Args:    sliceOf[ast.Expr](args),
		Rparen:  lastPos(c, ")"),
	}
	if where != nil {
		call.Where = where.(ast.Expr)
	}
	return call
}

func lastPos(c *current, s string) int {
	i := bytes.LastIndex(c.text, []byte(s))
	if i == -1 {
		panic(fmt.Sprintf("system error: character %s not found in %s", s, string(c.text)))
	}
	return c.pos.offset + i
}

func prepend(first, rest any) []any {
	return append([]any{first}, rest.([]any)...)
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

func nullableString(v any) string {
	if v == nil {
		return ""
	}
	return v.(string)
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

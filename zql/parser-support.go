package zql

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/brimsec/zq/ast"
	"github.com/brimsec/zq/pkg/joe"
	"github.com/mitchellh/mapstructure"
)

// ParseProc() is an entry point for use from external go code,
// mostly just a wrapper around Parse() that casts the return value.
func ParseProc(query string, opts ...Option) (ast.Proc, error) {
	parsed, err := Parse("", []byte(query), opts...)
	if err != nil {
		return nil, err
	}
	return ast.UnpackMap(nil, parsed)
}

func ParseExpression(expr string) (ast.Expression, error) {
	m, err := Parse("", []byte(expr), Entrypoint("Expr"))
	if err != nil {
		return nil, err
	}
	node := joe.Convert(m)
	ex, err := ast.UnpackExpression(node)
	if err != nil {
		return nil, err
	}
	c := &mapstructure.DecoderConfig{
		TagName: "json",
		Result:  ex,
		Squash:  true,
	}
	dec, err := mapstructure.NewDecoder(c)
	if err != nil {
		return nil, err
	}
	return ex, dec.Decode(m)
}

// MustParseProc is functionally the same as ParseProc but panics if an error
// is encountered.
func MustParseProc(query string) ast.Proc {
	proc, err := ParseProc(query)
	if err != nil {
		panic(err)
	}
	return proc
}

func makeChain(first interface{}, restIn interface{}, op string) interface{} {
	rest, ok := restIn.([]interface{})
	if !ok || len(rest) == 0 {
		return first
	}
	result := first
	for _, term := range rest {
		result = map[string]interface{}{
			"op":    op,
			"left":  result,
			"right": term,
		}
	}
	return result
}

func makeBinaryExprChain(first, rest interface{}) interface{} {
	ret := first
	for _, p := range rest.([]interface{}) {
		part := p.([]interface{})
		ret = map[string]interface{}{
			"op":       "BinaryExpr",
			"operator": part[0],
			"lhs":      ret,
			"rhs":      part[1],
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

func toLowerCase(in interface{}) interface{} {
	return strings.ToLower(in.(string))
}

func parseInt(v interface{}) interface{} {
	num := v.(string)
	i, err := strconv.Atoi(num)
	if err != nil {
		return nil
	}

	return i
}

func parseFloat(v interface{}) interface{} {
	num := v.(string)
	if f, err := strconv.ParseFloat(num, 10); err != nil {
		return f
	}

	return nil
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

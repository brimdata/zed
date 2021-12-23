package parser

import (
	"bytes"
	"fmt"
	"strconv"
)

func makeBinaryExprChain(first, rest interface{}) interface{} {
	ret := first
	for _, p := range rest.([]interface{}) {
		part := p.([]interface{})
		ret = map[string]interface{}{
			"kind": "BinaryExpr",
			"op":   part[0],
			"lhs":  ret,
			"rhs":  part[1],
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

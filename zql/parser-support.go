package zql

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

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
			"op":   "BinaryExpr",
			"kind": part[0],
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
	if f, err := strconv.ParseFloat(num, 64); err != nil {
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

func ImproveError(src string, e error) error {
	el, ok := e.(errList)
	if !ok || len(el) != 1 {
		return e
	}
	pe, ok := el[0].(*parserError)
	if !ok {
		return e
	}
	lineNo, colNo := pe.pos.line-1, pe.pos.col
	lines := strings.Split(src, "\n")
	if lineNo >= len(lines) {
		return e
	}
	var b strings.Builder
	if len(lines) == 1 {
		b.WriteString(fmt.Sprintf("error parsing Z at column %d:\n", colNo))
	} else {
		b.WriteString(fmt.Sprintf("error parsing Z at line %d, col %d:", lineNo+1, colNo))
	}
	b.WriteString(strings.Join(lines[:lineNo+1], "\n"))
	b.WriteByte('\n')
	colNo--
	for k := 0; k < colNo; k++ {
		if k >= colNo-4 && k != colNo-1 {
			b.WriteByte('=')
		} else {
			b.WriteByte(' ')
		}
	}
	b.WriteByte('^')
	b.WriteString(" ===")
	if lineNo+1 < len(lines) {
		b.WriteByte('\n')
		b.WriteString(strings.Join(lines[lineNo+1:], "\n"))
	}
	return errors.New(strings.TrimRight(b.String(), "\n"))
}

func ParseZ(src string) (interface{}, error) {
	p, err := Parse("", []byte(src))
	if err != nil {
		return nil, ImproveError(src, err)
	}
	return p, nil
}

func ParseZByRule(rule, src string) (interface{}, error) {
	p, err := Parse("", []byte(src), Entrypoint(rule))
	if err != nil {
		return nil, ImproveError(src, err)
	}
	return p, nil
}

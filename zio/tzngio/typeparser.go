package tzngio

import (
	"errors"
	"fmt"
	"strings"

	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

var ErrEmptyTypeList = errors.New("empty type list in set or union")

type TypeParser struct {
	zctx *zson.Context
}

func NewTypeParser(zctx *zson.Context) *TypeParser {
	return &TypeParser{
		zctx: zctx,
	}
}

func isIDChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '.'
}

func parseWord(in string) (string, string) {
	in = strings.TrimSpace(in)
	var off int
	for ; off < len(in); off++ {
		if !isIDChar(in[off]) {
			break
		}
	}
	if off == 0 {
		return "", ""
	}
	return in[off:], in[:off]
}

func (t *TypeParser) Parse(in string) (zng.Type, error) {
	rest, typ, err := t.parseType(in)
	if err == nil && rest != "" {
		err = zng.ErrTypeSyntax
	}
	return typ, err
}

func (t *TypeParser) parseType(in string) (string, zng.Type, error) {
	in = strings.TrimSpace(in)
	rest, word := parseWord(in)
	if word == "" {
		return "", nil, fmt.Errorf("unknown type: %s", in)
	}
	typ := zng.LookupPrimitive(word)
	if typ != nil {
		return rest, typ, nil
	}
	switch word {
	case "set":
		rest, typ, err := t.parseSetTypeBody(rest)
		if err != nil {
			return "", nil, err
		}
		return rest, typ, nil
	case "array":
		rest, typ, err := t.parseArrayTypeBody(rest)
		if err != nil {
			return "", nil, err
		}
		return rest, typ, nil
	case "record":
		return t.parseRecordTypeBody(rest)
	case "union":
		return t.parseUnionTypeBody(rest)
	case "enum":
		return t.parseEnumTypeBody(rest)
	case "map":
		return t.parseMapTypeBody(rest)
	}
	if typ := t.zctx.LookupTypeDef(word); typ != nil {
		return rest, typ, nil
	}
	return "", nil, fmt.Errorf("unknown type: %s", word)
}

func match(in, pattern string) (string, bool) {
	in = strings.TrimSpace(in)
	if strings.HasPrefix(in, pattern) {
		return in[len(pattern):], true
	}
	return in, false
}

// parseRecordTypeBody parses a list of record columns of the form "[field:type,...]".
func (t *TypeParser) parseRecordTypeBody(in string) (string, zng.Type, error) {
	in, ok := match(in, "[")
	if !ok {
		return "", nil, zng.ErrTypeSyntax
	}
	in, ok = match(in, "]")
	if ok {
		typ, err := t.zctx.LookupTypeRecord([]zng.Column{})
		if err != nil {
			return "", nil, err
		}
		return in, typ, nil
	}
	var columns []zng.Column
	for {
		// at top of loop, we have to have a field def either because
		// this is the first def or we found a comma and are expecting
		// another one.
		rest, col, err := t.parseColumn(in)
		if err != nil {
			return "", nil, err
		}
		columns = append(columns, col)
		rest, ok = match(rest, ",")
		if ok {
			in = rest
			continue
		}
		rest, ok = match(rest, "]")
		if !ok {
			return "", nil, zng.ErrTypeSyntax
		}
		typ, err := t.zctx.LookupTypeRecord(columns)
		if err != nil {
			return "", nil, err
		}
		return rest, typ, nil
	}
}

func (t *TypeParser) parseColumn(in string) (string, zng.Column, error) {
	in = strings.TrimSpace(in)
	rest, name, err := t.parseName(in)
	if err != nil {
		return "", zng.Column{}, err
	}
	var typ zng.Type
	rest, typ, err = t.parseType(rest)
	if err != nil {
		return "", zng.Column{}, err
	}
	if typ == nil {
		return "", zng.Column{}, zng.ErrTypeSyntax
	}
	return rest, zng.NewColumn(name, typ), nil
}

// parseTypeList parses a type list of the form "[type1,type2,type3]".
func (t *TypeParser) parseTypeList(in string) (string, []zng.Type, error) {
	rest, ok := match(in, "[")
	if !ok {
		return "", nil, zng.ErrTypeSyntax
	}
	if rest[0] == ']' {
		return "", nil, ErrEmptyTypeList
	}
	in = rest
	var types []zng.Type
	for {
		// at top of loop, we have to have a field def either because
		// this is the first def or we found a comma and are expecting
		// another one.
		rest, typ, err := t.parseType(in)
		if err != nil {
			return "", nil, err
		}
		types = append(types, typ)
		rest, ok = match(rest, ",")
		if ok {
			in = rest
			continue
		}
		rest, ok = match(rest, "]")
		if !ok {
			return "", nil, zng.ErrTypeSyntax
		}
		return rest, types, nil
	}
}

// parseSetTypeBody parses a set type body of the form "[type]" presuming the set
// keyword is already matched.
func (t *TypeParser) parseSetTypeBody(in string) (string, zng.Type, error) {
	rest, types, err := t.parseTypeList(in)
	if err != nil {
		return "", nil, err
	}
	if len(types) > 1 {
		return "", nil, fmt.Errorf("sets with multiple type parameters")
	}
	return rest, t.zctx.LookupTypeSet(types[0]), nil
}

// parseMapTypeBody parses a maap type body of the form "[type,type]" presuming
// the map keyword is already matched.
// The syntax is "map[keyType,valType]".
func (t *TypeParser) parseMapTypeBody(in string) (string, zng.Type, error) {
	rest, types, err := t.parseTypeList(in)
	if err != nil {
		return "", nil, err
	}
	if len(types) != 2 {
		return "", nil, fmt.Errorf("map type must have exactly two type parameters")
	}
	return rest, t.zctx.LookupTypeMap(types[0], types[1]), nil
}

// parseUnionTypeBody parses a set type body of the form
// "[type1,type2,...]" presuming the union keyword is already matched.
func (t *TypeParser) parseUnionTypeBody(in string) (string, zng.Type, error) {
	rest, types, err := t.parseTypeList(in)
	if err != nil {
		return "", nil, err
	}
	return rest, t.zctx.LookupTypeUnion(types), nil
}

// parse an array body type of the form "[type]"
func (t *TypeParser) parseArrayTypeBody(in string) (string, *zng.TypeArray, error) {
	rest, ok := match(in, "[")
	if !ok {
		return "", nil, zng.ErrTypeSyntax
	}
	var inner zng.Type
	var err error
	rest, inner, err = t.parseType(rest)
	if err != nil {
		return "", nil, err
	}
	rest, ok = match(rest, "]")
	if !ok {
		return "", nil, zng.ErrTypeSyntax
	}
	return rest, t.zctx.LookupTypeArray(inner), nil
}

// parse an array body type of the form "[type]"
func (t *TypeParser) parseEnumTypeBody(in string) (string, *zng.TypeEnum, error) {
	rest, ok := match(in, "[")
	if !ok {
		return "", nil, zng.ErrTypeSyntax
	}
	var symbols []string
	var err error
	rest, symbols, err = t.parseSymbols(rest)
	if err != nil {
		return "", nil, err
	}
	return rest, t.zctx.LookupTypeEnum(symbols), nil
}

// Parses a name up to the colon of the form "<id>:" or "[<tzng-strinig>]:"
func (c *TypeParser) parseName(in string) (string, string, error) {
	in = strings.TrimSpace(in)
	if in, ok := match(in, "["); ok {
		rbracket := strings.IndexByte(in, byte(']'))
		if rbracket < 0 {
			return "", "", zng.ErrTypeSyntax
		}
		name := strings.TrimSpace(in[:rbracket])
		rest := in[rbracket+1:]
		rest, ok := match(rest, ":")
		if !ok {
			return "", "", zng.ErrTypeSyntax
		}
		return rest, name, nil
	}
	colon := strings.IndexByte(in, byte(':'))
	if colon < 0 {
		return "", "", zng.ErrTypeSyntax
	}
	name := strings.TrimSpace(in[:colon])
	if !zng.IsIdentifier(name) {
		return "", "", zng.ErrTypeSyntax
	}
	return in[colon+1:], name, nil
}

func (c *TypeParser) parseSymbols(in string) (string, []string, error) {
	var symbols []string
	for {
		// at top of loop, we have to have a element def either because
		// this is the first def or we found a comma and are expecting
		// another one.
		rest, name, err := c.parseName(in)
		if err != nil {
			return "", nil, err
		}
		symbols = append(symbols, name)
		var ok bool
		rest, ok = match(rest, ",")
		if ok {
			in = rest
			continue
		}
		rest, ok = match(rest, "]")
		if !ok {
			return "", nil, zng.ErrTypeSyntax
		}
		return rest, symbols, nil
	}
}

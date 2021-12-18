package zeekio

import (
	"errors"
	"fmt"
	"strings"

	"github.com/brimdata/zed"
)

var ErrEmptyTypeList = errors.New("empty type list in set or union")

type TypeParser struct {
	zctx *zed.Context
}

func NewTypeParser(zctx *zed.Context) *TypeParser {
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

func (t *TypeParser) Parse(in string) (zed.Type, error) {
	rest, typ, err := t.parseType(in)
	if err == nil && rest != "" {
		err = zed.ErrTypeSyntax
	}
	return typ, err
}

func (t *TypeParser) parseType(in string) (string, zed.Type, error) {
	in = strings.TrimSpace(in)
	rest, word := parseWord(in)
	if word == "" {
		return "", nil, fmt.Errorf("unknown type: %s", in)
	}
	typ := zed.LookupPrimitive(word)
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
func (t *TypeParser) parseRecordTypeBody(in string) (string, zed.Type, error) {
	in, ok := match(in, "[")
	if !ok {
		return "", nil, zed.ErrTypeSyntax
	}
	in, ok = match(in, "]")
	if ok {
		typ, err := t.zctx.LookupTypeRecord([]zed.Column{})
		if err != nil {
			return "", nil, err
		}
		return in, typ, nil
	}
	var columns []zed.Column
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
			return "", nil, zed.ErrTypeSyntax
		}
		typ, err := t.zctx.LookupTypeRecord(columns)
		if err != nil {
			return "", nil, err
		}
		return rest, typ, nil
	}
}

func (t *TypeParser) parseColumn(in string) (string, zed.Column, error) {
	in = strings.TrimSpace(in)
	rest, name, err := t.parseName(in)
	if err != nil {
		return "", zed.Column{}, err
	}
	var typ zed.Type
	rest, typ, err = t.parseType(rest)
	if err != nil {
		return "", zed.Column{}, err
	}
	if typ == nil {
		return "", zed.Column{}, zed.ErrTypeSyntax
	}
	return rest, zed.NewColumn(name, typ), nil
}

// parseTypeList parses a type list of the form "[type1,type2,type3]".
func (t *TypeParser) parseTypeList(in string) (string, []zed.Type, error) {
	rest, ok := match(in, "[")
	if !ok {
		return "", nil, zed.ErrTypeSyntax
	}
	if rest[0] == ']' {
		return "", nil, ErrEmptyTypeList
	}
	in = rest
	var types []zed.Type
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
			return "", nil, zed.ErrTypeSyntax
		}
		return rest, types, nil
	}
}

// parseSetTypeBody parses a set type body of the form "[type]" presuming the set
// keyword is already matched.
func (t *TypeParser) parseSetTypeBody(in string) (string, zed.Type, error) {
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
func (t *TypeParser) parseMapTypeBody(in string) (string, zed.Type, error) {
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
func (t *TypeParser) parseUnionTypeBody(in string) (string, zed.Type, error) {
	rest, types, err := t.parseTypeList(in)
	if err != nil {
		return "", nil, err
	}
	return rest, t.zctx.LookupTypeUnion(types), nil
}

// parse an array body type of the form "[type]"
func (t *TypeParser) parseArrayTypeBody(in string) (string, *zed.TypeArray, error) {
	rest, ok := match(in, "[")
	if !ok {
		return "", nil, zed.ErrTypeSyntax
	}
	var inner zed.Type
	var err error
	rest, inner, err = t.parseType(rest)
	if err != nil {
		return "", nil, err
	}
	rest, ok = match(rest, "]")
	if !ok {
		return "", nil, zed.ErrTypeSyntax
	}
	return rest, t.zctx.LookupTypeArray(inner), nil
}

// parse an array body type of the form "[type]"
func (t *TypeParser) parseEnumTypeBody(in string) (string, *zed.TypeEnum, error) {
	rest, ok := match(in, "[")
	if !ok {
		return "", nil, zed.ErrTypeSyntax
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
			return "", "", zed.ErrTypeSyntax
		}
		name := strings.TrimSpace(in[:rbracket])
		rest := in[rbracket+1:]
		rest, ok := match(rest, ":")
		if !ok {
			return "", "", zed.ErrTypeSyntax
		}
		return rest, name, nil
	}
	colon := strings.IndexByte(in, byte(':'))
	if colon < 0 {
		return "", "", zed.ErrTypeSyntax
	}
	name := strings.TrimSpace(in[:colon])
	if !zed.IsIdentifier(name) {
		return "", "", zed.ErrTypeSyntax
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
			return "", nil, zed.ErrTypeSyntax
		}
		return rest, symbols, nil
	}
}

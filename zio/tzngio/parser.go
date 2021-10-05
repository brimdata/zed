package tzngio

import (
	"bytes"
	"encoding/base64"
	"errors"
	"net"
	"strconv"

	"github.com/brimdata/zed"
	astzed "github.com/brimdata/zed/compiler/ast/zed"
	"github.com/brimdata/zed/pkg/byteconv"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
	"golang.org/x/text/unicode/norm"
)

var (
	ErrUnterminated = errors.New("tzng syntax error: unterminated container")
	ErrSyntax       = errors.New("tzng syntax error")
)

type Parser struct {
	zcode.Builder
}

func NewParser() *Parser {
	return &Parser{*zcode.NewBuilder()}
}

// Parse decodes a zng value in text format using the type information
// in the descriptor.  Once parsed, the resulting zcode.Bytes has
// the nested data structure encoded independently of the data type.
func (p *Parser) Parse(typ *zed.TypeRecord, zng []byte) (zcode.Bytes, error) {
	p.Reset()
	if zng[0] != leftbracket {
		return nil, ErrSyntax
	}
	rest, err := p.ParseContainer(typ, zng)
	if err != nil {
		return nil, err
	}
	if len(rest) > 0 {
		return nil, ErrSyntax
	}
	return p.Bytes().ContainerBody()
}

const (
	semicolon    = byte(';')
	leftbracket  = byte('[')
	rightbracket = byte(']')
	backslash    = byte('\\')
)

// ParseContainer parses the given byte array representing a container
// in the zng format.
func (p *Parser) ParseContainer(typ zed.Type, b []byte) ([]byte, error) {
	realType := zed.AliasOf(typ)
	// This is hokey.
	var keyType, valType zed.Type
	if typ, ok := realType.(*zed.TypeMap); ok {
		keyType = typ.KeyType
		valType = typ.ValType
	}
	p.BeginContainer()
	// skip leftbracket
	b = b[1:]
	childType, columns := zed.ContainedType(realType)
	if childType == nil && columns == nil && keyType == nil {
		return nil, zed.ErrNotPrimitive
	}

	k := 0
	for {
		if len(b) == 0 {
			return nil, ErrUnterminated
		}
		if b[0] == rightbracket {
			if _, ok := realType.(*zed.TypeSet); ok {
				p.TransformContainer(zed.NormalizeSet)
			}
			if _, ok := realType.(*zed.TypeMap); ok {
				p.TransformContainer(zed.NormalizeMap)
			}
			p.EndContainer()
			return b[1:], nil
		}
		if columns != nil {
			if k >= len(columns) {
				return nil, &zed.RecordTypeError{Name: "<record>", Type: typ.String(), Err: zed.ErrExtraField}
			}
			childType = columns[k].Type
			k++
		}
		if keyType != nil {
			if (k & 1) == 0 {
				childType = keyType
			} else {
				childType = valType
			}
			k++
		}
		rest, err := p.ParseField(childType, b)
		if err != nil {
			return nil, err
		}
		b = rest
	}
}

// ParseField parses the given byte array representing any value
// in the zng format.
func (p *Parser) ParseField(typ zed.Type, b []byte) ([]byte, error) {
	realType := zed.AliasOf(typ)
	if len(b) >= 2 && b[0] == '-' && b[1] == ';' {
		if zed.IsContainerType(realType) {
			p.AppendContainer(nil)
		} else {
			p.AppendPrimitive(nil)
		}
		return b[2:], nil
	}
	if utyp, ok := realType.(*zed.TypeUnion); ok {
		childType, selector, bb, err := parseUnion(utyp, b)
		if err != nil {
			return nil, err
		}
		b = bb
		p.BeginContainer()
		defer p.EndContainer()
		p.AppendPrimitive(zed.EncodeInt(int64(selector)))
		return p.ParseField(childType, b)
	}
	if b[0] == leftbracket {
		return p.ParseContainer(typ, b)
	}
	if zed.IsContainerType(realType) {
		return nil, zed.ErrNotContainer
	}

	// We don't actually need to handle escapes here, type.Parse()
	// will take care of that.  The important thing is just figuring
	// out the proper boundary between individual records (skipping
	// over an escaped semicolon without being tricked by something
	// like \\; which could be an escaped backslash followed by a
	// real semicolon)
	i := 0
	for ; i < len(b); i++ {
		if b[i] == semicolon {
			break
		}
		if b[i] == backslash {
			i++
		}
	}
	if i == len(b) {
		return nil, ErrUnterminated
	}

	zv, err := ParseValue(realType, b[:i])
	if err != nil {
		return nil, err
	}
	p.AppendPrimitive(zv)
	return b[i+1:], nil
}

// parseUnion takes a union type and a tzng encoding of a value of that type
// and returns the concrete type of the value, its selector, and the value encoding.
func parseUnion(u *zed.TypeUnion, in []byte) (zed.Type, int, []byte, error) {
	c := bytes.IndexByte(in, ':')
	if c < 0 {
		return nil, -1, nil, ErrBadValue
	}
	selector, err := strconv.Atoi(string(in[0:c]))
	if err != nil {
		return nil, -1, nil, err
	}
	typ, err := u.Type(selector)
	if err != nil {
		return nil, -1, nil, err
	}
	return typ, selector, in[c+1:], nil
}

func ParseContainer(typ zed.Type, in []byte) (zcode.Bytes, error) {
	p := NewParser()
	_, err := p.ParseContainer(typ, in)
	if err != nil {
		return nil, err
	}
	return p.Bytes().ContainerBody()
}

func ParseMap(t *zed.TypeMap, in []byte) (zcode.Bytes, error) {
	return ParseContainer(t, in)
}

func ParseRecord(t *zed.TypeRecord, in []byte) (zcode.Bytes, error) {
	return ParseContainer(t, in)
}

func ParseSet(t *zed.TypeSet, in []byte) (zcode.Bytes, error) {
	return ParseContainer(t, in)
}

func ParseUnion(t *zed.TypeUnion, in []byte) (zcode.Bytes, error) {
	return ParseContainer(t, in)
}

func ParseBstring(in []byte) (zcode.Bytes, error) {
	normalized := norm.NFC.Bytes(zed.UnescapeBstring(in))
	return normalized, nil
}

func ParseString(in []byte) (zcode.Bytes, error) {
	normalized := norm.NFC.Bytes(UnescapeString(in))
	return normalized, nil
}

func ParseBool(in []byte) (zcode.Bytes, error) {
	b, err := byteconv.ParseBool(in)
	if err != nil {
		return nil, err
	}
	return zed.EncodeBool(b), nil
}

func ParseBytes(in []byte) (zcode.Bytes, error) {
	s := string(in)
	if s == "" {
		return []byte{}, nil
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return zcode.Bytes(b), nil
}

func ParseDuration(in []byte) (zcode.Bytes, error) {
	d, err := nano.Parse(in) // zeek-style, full 64-bit ns fractional number
	if err != nil {
		return nil, err
	}
	return zed.EncodeDuration(nano.Duration(d)), nil
}

func ParseIP(in []byte) (zcode.Bytes, error) {
	ip, err := byteconv.ParseIP(in)
	if err != nil {
		return nil, err
	}
	return zed.EncodeIP(ip), nil
}

func ParseNet(in []byte) (zcode.Bytes, error) {
	_, subnet, err := net.ParseCIDR(string(in))
	if err != nil {
		return nil, err
	}
	return zed.EncodeNet(subnet), nil
}

func ParseTime(in []byte) (zcode.Bytes, error) {
	ts, err := nano.Parse(in)
	if err != nil {
		return nil, err
	}
	return zed.EncodeTime(ts), nil
}

func ParseValue(typ zed.Type, in []byte) (zcode.Bytes, error) {
	switch typ := typ.(type) {
	case *zed.TypeAlias:
		return ParseValue(typ.Type, in)
	case *zed.TypeRecord:
		return ParseRecord(typ, in)
	case *zed.TypeArray, *zed.TypeSet, *zed.TypeUnion, *zed.TypeMap:
		return ParseContainer(typ, in)
	case *zed.TypeEnum:
		return ParseValue(zed.TypeUint64, in)
	case *zed.TypeOfBool:
		return ParseBool(in)
	case *zed.TypeOfBytes:
		return ParseBytes(in)
	case *zed.TypeOfBstring:
		return ParseBstring(in)
	case *zed.TypeOfDuration:
		return ParseDuration(in)
	case *zed.TypeOfIP:
		return ParseIP(in)
	case *zed.TypeOfNet:
		return ParseNet(in)
	case *zed.TypeOfString:
		return ParseString(in)
	case *zed.TypeOfTime:
		return ParseTime(in)
	default:
		primitive := astzed.Primitive{
			Kind: "Primitive",
			Type: typ.String(),
			Text: string(in),
		}
		zv, err := zson.ParsePrimitive(primitive.Type, primitive.Text)
		if err != nil {
			return nil, err
		}
		return zv.Bytes, nil
	}
}

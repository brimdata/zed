package zngio

import (
	"errors"

	"github.com/mccanne/zq/zcode"
	"github.com/mccanne/zq/zng"
)

var (
	ErrUnterminated = errors.New("zng syntax error: unterminated container")
	ErrSyntax       = errors.New("zng syntax error")
)

type Parser struct {
	builder *zcode.Builder
}

func NewParser() *Parser {
	return &Parser{
		builder: zcode.NewBuilder(),
	}
}

// Parse decodes a zng value in text format using the type information
// in the descriptor.  Once parsed, the resulting zcode.Bytes has
// the nested data structure encoded independently of the data type.
func (p *Parser) Parse(typ *zng.TypeRecord, zng []byte) (zcode.Bytes, error) {
	builder := p.builder
	builder.Reset()
	if zng[0] != leftbracket {
		return nil, ErrSyntax
	}
	rest, err := zngParseContainer(builder, typ, zng)
	if err != nil {
		return nil, err
	}
	if len(rest) > 0 {
		return nil, ErrSyntax
	}
	return builder.Bytes().ContainerBody()
}

const (
	semicolon    = byte(';')
	leftbracket  = byte('[')
	rightbracket = byte(']')
	backslash    = byte('\\')
)

// zngParseContainer parses the given byte array representing a container
// in the zng format.
func zngParseContainer(builder *zcode.Builder, typ zng.Type, b []byte) ([]byte, error) {
	builder.BeginContainer()
	// skip leftbracket
	b = b[1:]
	childType, columns := zng.ContainedType(typ)
	if childType == nil && columns == nil {
		return nil, zng.ErrNotPrimitive
	}
	k := 0
	for {
		if len(b) == 0 {
			return nil, ErrUnterminated
		}
		if b[0] == rightbracket {
			builder.EndContainer()
			return b[1:], nil
		}
		if columns != nil {
			if k >= len(columns) {
				return nil, &zng.RecordTypeError{Name: "<record>", Type: typ.String(), Err: zng.ErrExtraField}
			}
			childType = columns[k].Type
			k++
		}
		rest, err := zngParseField(builder, childType, b)
		if err != nil {
			return nil, err
		}
		b = rest
	}
}

// zngParseField parses the given byte array representing any value
// in the zng format.
func zngParseField(builder *zcode.Builder, typ zng.Type, b []byte) ([]byte, error) {
	var err error
	var index int
	if len(b) >= 2 && b[0] == '-' && b[1] == ';' {
		if zng.IsContainerType(typ) || zng.IsUnionType(typ) {
			builder.AppendContainer(nil)
		} else {
			builder.AppendPrimitive(nil)
		}
		return b[2:], nil
	}
	if utyp, ok := typ.(*zng.TypeUnion); ok {
		b, typ, index, err = utyp.SplitZng(b)
		if err != nil {
			return nil, err
		}
		builder.BeginContainer()
		defer builder.EndContainer()
		var a [8]byte
		n := zcode.EncodeCountedVarint(a[:], int64(index))
		builder.AppendPrimitive(a[:n])
	}
	if b[0] == leftbracket {
		return zngParseContainer(builder, typ, b)
	}
	if zng.IsContainerType(typ) {
		return nil, zng.ErrNotContainer
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

	zv, err := typ.Parse(b[:i])
	if err != nil {
		return nil, err
	}
	builder.AppendPrimitive(zv)
	return b[i+1:], nil
}

package zbuf

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/zcode"
	"github.com/mccanne/zq/zng"
)

// EncodeZvals builds a raw value from a descriptor and zvals.
func EncodeZvals(d *Descriptor, vals []zcode.Bytes) (zcode.Bytes, error) {
	if nv, nc := len(vals), len(d.Type.Columns); nv != nc {
		return nil, fmt.Errorf("got %d values (%q), expected %d (%q)", nv, vals, nc, d.Type.Columns)

	}
	var raw zcode.Bytes
	for _, val := range vals {
		raw = zcode.AppendPrimitive(raw, val)
	}
	return raw, nil
}

func NewRawAndTsFromZeekTSV(builder *zcode.Builder, d *Descriptor, path []byte, data []byte) (zcode.Bytes, zng.Value, error) {
	builder.Reset()
	columns := d.Type.Columns
	col := 0
	tsVal := zng.Value{}
	if path != nil {
		if columns[0].Name != "_path" {
			return nil, zng.Value{}, errors.New("no _path in column 0")
		}
		builder.AppendPrimitive(path)
		col++
	}

	const separator = '\t'
	const setSeparator = ','
	const emptyContainer = "(empty)"
	var start int
	nestedCol := 0
	handleVal := func(val []byte) error {
		if col >= len(columns) {
			return errors.New("too many values")
		}

		typ := columns[col].Type
		recType, isRec := typ.(*zng.TypeRecord)
		if isRec {
			if nestedCol == 0 {
				builder.BeginContainer()
			}
			typ = recType.Columns[nestedCol].Type
		}

		if len(val) == 1 && val[0] == '-' {
			switch typ.(type) {
			case *zng.TypeSet, *zng.TypeVector:
				builder.AppendContainer(nil)
			default:
				builder.AppendPrimitive(nil)
			}
		} else {
			switch typ.(type) {
			case *zng.TypeSet, *zng.TypeVector:
				inner := zng.InnerType(typ)
				builder.BeginContainer()
				if bytes.Compare(val, []byte(emptyContainer)) != 0 {
					cstart := 0
					for i, ch := range val {
						if ch == setSeparator {
							zv, err := inner.Parse(zng.Unescape(val[cstart:i]))
							if err != nil {
								return err
							}
							builder.AppendPrimitive(zv)
							cstart = i + 1
						}
					}
					zv, err := inner.Parse(zng.Unescape(val[cstart:]))
					if err != nil {
						return err
					}
					builder.AppendPrimitive(zv)
				}
				builder.EndContainer()
			default:
				// regular (non-container) value
				zv, err := typ.Parse(zng.Unescape(val))
				if err != nil {
					return err
				}
				//XXX pulling out ts field should be done outside
				// of this routine... this is severe bit rot
				if columns[col].Name == "ts" {
					_, err := zng.DecodeTime(zv)
					if err != nil {
						return err
					}
					tsVal = zng.Value{zng.TypeTime, zv}
				}
				builder.AppendPrimitive(zv)
			}
		}

		if isRec {
			nestedCol++
			if nestedCol == len(recType.Columns) {
				builder.EndContainer()
				nestedCol = 0
				col++
			}
		} else {
			col++
		}
		return nil
	}

	for i, c := range data {
		if c == separator {
			err := handleVal(data[start:i])
			if err != nil {
				return nil, zng.Value{}, err
			}
			start = i + 1
		}
	}
	err := handleVal(data[start:])
	if err != nil {
		return nil, zng.Value{}, err
	}

	if col != len(d.Type.Columns) {
		return nil, zng.Value{}, errors.New("too few values")
	}
	return builder.Bytes(), tsVal, nil
}

func NewRawAndTsFromZeekValues(d *Descriptor, tsCol int, vals [][]byte) (zcode.Bytes, nano.Ts, error) {
	if nv, nc := len(vals), len(d.Type.Columns); nv != nc {
		// Don't pass vals to fmt.Errorf or it will escape to the heap.
		return nil, 0, fmt.Errorf("got %d values, expected %d", nv, nc)
	}
	n := 2 // Estimate for descriptor ID uvarint.
	for _, v := range vals {
		n += len(v) + 1 // Estimate for zval and its length uvarint.
	}
	raw := make([]byte, 0, n)
	var ts nano.Ts
	for i, val := range vals {
		var err error
		if i == tsCol {
			ts, err = nano.Parse(val)
			if err != nil {
				return nil, 0, err
			}
		}
		raw = appendZvalFromZeek(raw, d.Type.Columns[i].Type, val)
	}
	return raw, ts, nil
}

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
func (p *Parser) Parse(d *Descriptor, zng []byte) (zcode.Bytes, error) {
	builder := p.builder
	builder.Reset()
	if zng[0] != leftbracket {
		return nil, ErrSyntax
	}
	rest, err := zngParseContainer(builder, d.Type, zng)
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

// zngParseContainer() parses the given byte array representing a container
// in the zng format.
// If there is no error, the first two return values are:
//  1. an array of zvals corresponding to the indivdiual elements
//  2. the passed-in byte array advanced past all the data that was parsed.
func zngParseContainer(builder *zcode.Builder, typ zng.Type, b []byte) ([]byte, error) {
	builder.BeginContainer()
	// skip leftbracket
	b = b[1:]
	childType, columns := zng.ContainedType(typ)
	if childType == nil && columns == nil {
		return nil, ErrNotPrimitive
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
				return nil, &RecordTypeError{Name: "<record>", Type: typ.String(), Err: ErrExtraField}
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

// zngParseField() parses the given bye array representing any value
// in the zng format.
func zngParseField(builder *zcode.Builder, typ zng.Type, b []byte) ([]byte, error) {
	if b[0] == leftbracket {
		return zngParseContainer(builder, typ, b)
	}
	if len(b) >= 2 && b[0] == '-' && b[1] == ';' {
		if zng.IsContainerType(typ) {
			builder.AppendContainer(nil)
		} else {
			builder.AppendPrimitive(nil)
		}
		return b[2:], nil
	}
	to := 0
	from := 0
	for {
		if from >= len(b) {
			return nil, ErrUnterminated
		}
		switch b[from] {
		case semicolon:
			if zng.IsContainerType(typ) {
				return nil, ErrNotContainer
			}
			zv, err := typ.Parse(zng.Unescape(b[:to]))
			if err != nil {
				return nil, err
			}
			builder.AppendPrimitive(zv)
			return b[from+1:], nil
		case backslash:
			e, n := zng.ParseEscape(b[from:])
			if n == 0 {
				panic("zng.ParseEscape returned 0")
			}
			b[to] = e
			from += n
		default:
			b[to] = b[from]
			from++
		}
		to++
	}
}

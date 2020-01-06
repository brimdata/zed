package zng

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zval"
)

// EncodeZvals builds a raw value from a descriptor and zvals.
func EncodeZvals(d *Descriptor, vals []zval.Encoding) (zval.Encoding, error) {
	if nv, nc := len(vals), len(d.Type.Columns); nv != nc {
		return nil, fmt.Errorf("got %d values (%q), expected %d (%q)", nv, vals, nc, d.Type.Columns)

	}
	var raw zval.Encoding
	for _, val := range vals {
		raw = zval.AppendValue(raw, val)
	}
	return raw, nil
}

func NewRawAndTsFromZeekTSV(builder *zval.Builder, d *Descriptor, path []byte, data []byte) (zval.Encoding, zeek.Value, error) {
	builder.Reset()
	columns := d.Type.Columns
	col := 0
	var tsVal zeek.Value
	tsVal = &zeek.Unset{}
	if path != nil {
		if columns[0].Name != "_path" {
			return nil, nil, errors.New("no _path in column 0")
		}
		builder.Append(path, false)
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
		recType, isRec := typ.(*zeek.TypeRecord)
		if isRec {
			if nestedCol == 0 {
				builder.BeginContainer()
			}
			typ = recType.Columns[nestedCol].Type
		}

		if len(val) == 1 && val[0] == '-' {
			switch typ.(type) {
			case *zeek.TypeSet, *zeek.TypeVector:
				builder.AppendUnsetContainer()
			default:
				builder.AppendUnsetValue()
			}
		} else {
			switch typ.(type) {
			case *zeek.TypeSet, *zeek.TypeVector:
				inner := zeek.InnerType(typ)
				builder.BeginContainer()
				if bytes.Compare(val, []byte(emptyContainer)) != 0 {
					cstart := 0
					for i, ch := range val {
						if ch == setSeparator {
							zv, err := inner.Parse(zeek.Unescape(val[cstart:i]))
							if err != nil {
								return err
							}
							builder.Append(zv, false)
							cstart = i + 1
						}
					}
					zv, err := inner.Parse(zeek.Unescape(val[cstart:]))
					if err != nil {
						return err
					}
					builder.Append(zv, false)
				}
				builder.EndContainer()
			default:
				// regular (non-container) value
				zv, err := typ.Parse(zeek.Unescape(val))
				if err != nil {
					return err
				}
				if columns[col].Name == "ts" {
					tt := zeek.TypeOfTime{}
					tsVal, err = tt.New(zv)
					if err != nil {
						return err
					}
				}
				builder.Append(zv, false)
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
				return nil, nil, err
			}
			start = i + 1
		}
	}
	err := handleVal(data[start:])
	if err != nil {
		return nil, nil, err
	}

	if col != len(d.Type.Columns) {
		return nil, nil, errors.New("too few values")
	}
	return builder.Encode(), tsVal, nil
}

func NewRawAndTsFromZeekValues(d *Descriptor, tsCol int, vals [][]byte) (zval.Encoding, nano.Ts, error) {
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
	builder *zval.Builder
}

func NewParser() *Parser {
	return &Parser{
		builder: zval.NewBuilder(),
	}
}

// Parse decodes a zng value in text format using the type information
// in the descriptor.  Once parsed, the resulting zval.Encoding has
// the nested data structure encoded independently of the data type.
func (p *Parser) Parse(d *Descriptor, zng []byte) (zval.Encoding, error) {
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
	return builder.Encode().Body()
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
func zngParseContainer(builder *zval.Builder, typ zeek.Type, b []byte) ([]byte, error) {
	builder.BeginContainer()
	// skip leftbracket
	b = b[1:]
	childType, columns := zeek.ContainedType(typ)
	if childType == nil && columns == nil {
		return nil, ErrNotScalar
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
func zngParseField(builder *zval.Builder, typ zeek.Type, b []byte) ([]byte, error) {
	if b[0] == leftbracket {
		return zngParseContainer(builder, typ, b)
	}
	if len(b) >= 2 && b[0] == '-' && b[1] == ';' {
		if zeek.IsContainerType(typ) {
			builder.AppendUnsetContainer()
		} else {
			builder.AppendUnsetValue()
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
			if zeek.IsContainerType(typ) {
				return nil, ErrNotContainer
			}
			zv, err := typ.Parse(zeek.Unescape(b[:to]))
			if err != nil {
				return nil, err
			}
			builder.Append(zv, false)
			return b[from+1:], nil
		case backslash:
			e, n := zeek.ParseEscape(b[from:])
			if n == 0 {
				panic("zeek.ParseEscape returned 0")
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

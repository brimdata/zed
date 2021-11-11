package zeekio

import (
	"bytes"
	"errors"
	"net"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/byteconv"
	"github.com/brimdata/zed/pkg/nano"
	"github.com/brimdata/zed/zcode"
	"golang.org/x/text/unicode/norm"
)

type builder struct {
	zcode.Builder
	buf             []byte
	fields          [][]byte
	reorderedFields [][]byte
}

func (b *builder) build(typ *zed.TypeRecord, sourceFields []int, path []byte, data []byte) (*zed.Value, error) {
	b.Reset()
	b.Grow(len(data))
	columns := typ.Columns
	if path != nil {
		if columns[0].Name != "_path" {
			return nil, errors.New("no _path in column 0")
		}
		columns = columns[1:]
		b.AppendPrimitive(path)
	}
	b.fields = b.fields[:0]
	var start int

	const separator = '\t'

	for i, c := range data {
		if c == separator {
			b.fields = append(b.fields, data[start:i])
			start = i + 1
		}
	}
	b.fields = append(b.fields, data[start:])
	if actual, expected := len(b.fields), len(sourceFields); actual > expected {
		return nil, errors.New("too many values")
	} else if actual < expected {
		return nil, errors.New("too few values")
	}
	b.reorderedFields = b.reorderedFields[:0]
	for _, s := range sourceFields {
		b.reorderedFields = append(b.reorderedFields, b.fields[s])
	}
	leftoverFields, err := b.appendColumns(columns, b.reorderedFields)
	if err != nil {
		return nil, err
	}
	if len(leftoverFields) != 0 {
		return nil, errors.New("too many values")
	}
	return zed.NewValue(typ, b.Bytes()), nil
}

func (b *builder) appendColumns(columns []zed.Column, fields [][]byte) ([][]byte, error) {
	const setSeparator = ','
	const emptyContainer = "(empty)"
	for _, c := range columns {
		if len(fields) == 0 {
			return nil, errors.New("too few values")
		}
		switch typ := c.Type.(type) {
		case *zed.TypeArray, *zed.TypeSet:
			val := fields[0]
			fields = fields[1:]
			if string(val) == "-" {
				b.AppendContainer(nil)
				continue
			}
			b.BeginContainer()
			if bytes.Equal(val, []byte(emptyContainer)) {
				b.EndContainer()
				continue
			}
			inner := zed.InnerType(typ)
			var cstart int
			for i, ch := range val {
				if ch == setSeparator {
					if err := b.appendPrimitive(inner, val[cstart:i]); err != nil {
						return nil, err
					}
					cstart = i + 1
				}
			}
			if err := b.appendPrimitive(inner, val[cstart:]); err != nil {
				return nil, err
			}
			if _, ok := typ.(*zed.TypeSet); ok {
				b.TransformContainer(zed.NormalizeSet)
			}
			b.EndContainer()
		case *zed.TypeRecord:
			b.BeginContainer()
			var err error
			if fields, err = b.appendColumns(typ.Columns, fields); err != nil {
				return nil, err
			}
			b.EndContainer()
		default:
			if err := b.appendPrimitive(c.Type, fields[0]); err != nil {
				return nil, err
			}
			fields = fields[1:]
		}
	}
	return fields, nil
}

func (b *builder) appendPrimitive(typ zed.Type, val []byte) error {
	if string(val) == "-" {
		b.AppendPrimitive(nil)
		return nil
	}
	switch typ.ID() {
	case zed.IDInt64:
		v, err := byteconv.ParseInt64(val)
		if err != nil {
			return err
		}
		b.buf = zed.AppendInt(b.buf[:0], v)
	case zed.IDUint16:
		// Zeek's port type is aliased to uint16.
		v, err := byteconv.ParseUint16(val)
		if err != nil {
			return err
		}
		b.buf = zed.AppendUint(b.buf[:0], uint64(v))
	case zed.IDUint64:
		v, err := byteconv.ParseUint64(val)
		if err != nil {
			return err
		}
		b.buf = zed.AppendUint(b.buf[:0], v)
	case zed.IDDuration:
		v, err := nano.Parse(val) // zeek-style fractional ts
		if err != nil {
			return err
		}
		b.buf = zed.AppendDuration(b.buf[:0], nano.Duration(v))
	case zed.IDTime:
		v, err := nano.Parse(val)
		if err != nil {
			return err
		}
		b.buf = zed.AppendTime(b.buf[:0], v)
	case zed.IDFloat64:
		v, err := byteconv.ParseFloat64(val)
		if err != nil {
			return err
		}
		b.buf = zed.AppendFloat64(b.buf[:0], v)
	case zed.IDBool:
		v, err := byteconv.ParseBool(val)
		if err != nil {
			return err
		}
		b.buf = zed.AppendBool(b.buf[:0], v)
	case zed.IDString:
		// Zeek's enum type is aliased to string.
		b.AppendPrimitive(norm.NFC.Bytes(val))
		return nil
	case zed.IDBstring:
		b.AppendPrimitive(norm.NFC.Bytes(zed.UnescapeBstring(val)))
		return nil
	case zed.IDIP:
		v, err := byteconv.ParseIP(val)
		if err != nil {
			return err
		}
		b.buf = zed.AppendIP(b.buf[:0], v)
	case zed.IDNet:
		_, v, err := net.ParseCIDR(string(val))
		if err != nil {
			return err
		}
		b.buf = zed.AppendNet(b.buf[:0], v)
	default:
		panic(typ)
	}
	b.AppendPrimitive(b.buf)
	return nil
}

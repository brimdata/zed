package zeekio

import (
	"bytes"
	"errors"
	"net"

	"github.com/brimdata/zq/pkg/byteconv"
	"github.com/brimdata/zq/pkg/nano"
	"github.com/brimdata/zq/zcode"
	"github.com/brimdata/zq/zio/tzngio"
	"github.com/brimdata/zq/zng"
)

type builder struct {
	zcode.Builder
	buf             []byte
	fields          [][]byte
	reorderedFields [][]byte
}

func (b *builder) build(typ *zng.TypeRecord, sourceFields []int, path []byte, data []byte) (*zng.Record, error) {
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
	return zng.NewRecord(typ, b.Bytes()), nil
}

func (b *builder) appendColumns(columns []zng.Column, fields [][]byte) ([][]byte, error) {
	const setSeparator = ','
	const emptyContainer = "(empty)"
	for _, c := range columns {
		if len(fields) == 0 {
			return nil, errors.New("too few values")
		}
		switch typ := c.Type.(type) {
		case *zng.TypeArray, *zng.TypeSet:
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
			inner := zng.InnerType(typ)
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
			if _, ok := typ.(*zng.TypeSet); ok {
				b.TransformContainer(zng.NormalizeSet)
			}
			b.EndContainer()
		case *zng.TypeRecord:
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

func (b *builder) appendPrimitive(typ zng.Type, val []byte) error {
	if string(val) == "-" {
		b.AppendPrimitive(nil)
		return nil
	}
	switch typ.ID() {
	case zng.IdInt64:
		v, err := byteconv.ParseInt64(val)
		if err != nil {
			return err
		}
		b.buf = zng.AppendInt(b.buf[:0], v)
	case zng.IdUint16:
		// Zeek's port type is aliased to uint16.
		v, err := byteconv.ParseUint16(val)
		if err != nil {
			return err
		}
		b.buf = zng.AppendUint(b.buf[:0], uint64(v))
	case zng.IdUint64:
		v, err := byteconv.ParseUint64(val)
		if err != nil {
			return err
		}
		b.buf = zng.AppendUint(b.buf[:0], v)
	case zng.IdDuration:
		v, err := nano.Parse(val) // zeek-style fractional ts
		if err != nil {
			return err
		}
		b.buf = zng.AppendDuration(b.buf[:0], nano.Duration(v))
	case zng.IdTime:
		v, err := nano.Parse(val)
		if err != nil {
			return err
		}
		b.buf = zng.AppendTime(b.buf[:0], v)
	case zng.IdFloat64:
		v, err := byteconv.ParseFloat64(val)
		if err != nil {
			return err
		}
		b.buf = zng.AppendFloat64(b.buf[:0], v)
	case zng.IdBool:
		v, err := byteconv.ParseBool(val)
		if err != nil {
			return err
		}
		b.buf = zng.AppendBool(b.buf[:0], v)
	case zng.IdString:
		// Zeek's enum type is aliased to string.
		zb, err := tzngio.ParseString(val)
		if err != nil {
			return err
		}
		b.AppendPrimitive(zb)
		return nil
	case zng.IdBstring:
		zb, err := tzngio.ParseBstring(val)
		if err != nil {
			return err
		}
		b.AppendPrimitive(zb)
		return nil
	case zng.IdIP:
		v, err := byteconv.ParseIP(val)
		if err != nil {
			return err
		}
		b.buf = zng.AppendIP(b.buf[:0], v)
	case zng.IdNet:
		_, v, err := net.ParseCIDR(string(val))
		if err != nil {
			return err
		}
		b.buf = zng.AppendNet(b.buf[:0], v)
	default:
		panic(typ)
	}
	b.AppendPrimitive(b.buf)
	return nil
}

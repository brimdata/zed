package zeekio

import (
	"bytes"
	"errors"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

type builder struct {
	zcode.Builder
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
	if len(b.fields) > len(sourceFields) {
		return nil, errors.New("too many values")
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
	zv, err := typ.Parse(val)
	if err != nil {
		return err
	}
	b.AppendPrimitive(zv)
	return nil
}

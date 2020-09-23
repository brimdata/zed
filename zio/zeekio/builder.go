package zeekio

import (
	"bytes"
	"errors"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

type builder struct {
	zcode.Builder
}

func (b *builder) build(typ *zng.TypeRecord, sourceFields []int, path []byte, data []byte) (*zng.Record, error) {
	b.Reset()
	columns := typ.Columns
	if path != nil {
		if columns[0].Name != "_path" {
			return nil, errors.New("no _path in column 0")
		}
		columns = columns[1:]
		b.AppendPrimitive(path)
	}
	var fields [][]byte
	var start int

	const separator = '\t'

	for i, c := range data {
		if c == separator {
			fields = append(fields, data[start:i])
			start = i + 1
		}
	}
	fields = append(fields, data[start:])
	if len(fields) > len(sourceFields) {
		return nil, errors.New("too many values")
	}
	var fields2 [][]byte
	for _, s := range sourceFields {
		fields2 = append(fields2, fields[s])
	}
	fields, err := b.appendColumns(columns, fields2)
	if err != nil {
		return nil, err
	}
	if len(fields) != 0 {
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

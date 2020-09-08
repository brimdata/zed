package zeekio

import (
	"bytes"
	"errors"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

func buildRecordFromZeekTSV(builder *zcode.Builder, typ *zng.TypeRecord, sourceFields []int, path []byte, data []byte) (zcode.Bytes, error) {
	columns := typ.Columns
	if path != nil {
		if columns[0].Name != "_path" {
			return nil, errors.New("no _path in column 0")
		}
		columns = columns[1:]
		builder.AppendPrimitive(path)
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
	fields, err := appendRecordFromZeekTSV(builder, columns, fields2)
	if err != nil {
		return nil, err
	}
	if len(fields) != 0 {
		return nil, errors.New("too many values")
	}

	return builder.Bytes(), nil
}

func appendRecordFromZeekTSV(builder *zcode.Builder, columns []zng.Column, fields [][]byte) ([][]byte, error) {
	const setSeparator = ','
	const emptyContainer = "(empty)"

	appendVal := func(val []byte, typ zng.Type) error {
		if string(val) == "-" {
			builder.AppendPrimitive(nil)
			return nil
		}
		zv, err := typ.Parse(val)
		if err != nil {
			return err
		}
		builder.AppendPrimitive(zv)
		return nil
	}

	handleVal := func(val []byte, typ zng.Type) error {
		switch typ.(type) {
		case *zng.TypeSet, *zng.TypeArray:
			if string(val) == "-" {
				builder.AppendContainer(nil)
				break
			}
			inner := zng.InnerType(typ)
			builder.BeginContainer()
			if bytes.Equal(val, []byte(emptyContainer)) {
				builder.EndContainer()
				break
			}
			cstart := 0
			for i, ch := range val {
				if ch == setSeparator {
					if err := appendVal(val[cstart:i], inner); err != nil {
						return err
					}
					cstart = i + 1
				}
			}
			if err := appendVal(val[cstart:], inner); err != nil {
				return err
			}
			if _, ok := typ.(*zng.TypeSet); ok {
				builder.TransformContainer(zng.NormalizeSet)
			}
			builder.EndContainer()
		default:
			if err := appendVal(val, typ); err != nil {
				return err
			}
		}
		return nil
	}

	c := 0
	for c < len(columns) {
		if len(fields) == 0 {
			return nil, errors.New("too few values")
		}

		typ := columns[c].Type
		if recType, isRec := typ.(*zng.TypeRecord); isRec {
			builder.BeginContainer()
			var err error
			if fields, err = appendRecordFromZeekTSV(builder, recType.Columns, fields); err != nil {
				return nil, err
			}
			builder.EndContainer()
		} else {
			if err := handleVal(fields[0], typ); err != nil {
				return nil, err
			}
			fields = fields[1:]
		}
		c++
	}

	return fields, nil
}

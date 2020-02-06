package zbuf

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
)

func NewRawFromZeekTSV(builder *zcode.Builder, typ *zng.TypeRecord, path []byte, data []byte) (zcode.Bytes, error) {
	builder.Reset()
	columns := typ.Columns
	col := 0
	if path != nil {
		if columns[0].Name != "_path" {
			return nil, errors.New("no _path in column 0")
		}
		builder.AppendPrimitive(path)
		col++
	}

	const separator = '\t'
	const setSeparator = ','
	const emptyContainer = "(empty)"
	var start int
	nestedCol := 0
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

		if isRec {
			nestedCol++
			if nestedCol != len(recType.Columns) {
				return nil
			}
			builder.EndContainer()
			nestedCol = 0
		}
		col++
		return nil
	}

	for i, c := range data {
		if c == separator {
			err := handleVal(data[start:i])
			if err != nil {
				return nil, err
			}
			start = i + 1
		}
	}
	err := handleVal(data[start:])
	if err != nil {
		return nil, err
	}

	if col != len(typ.Columns) {
		return nil, errors.New("too few values")
	}
	return builder.Bytes(), nil
}

func NewRawAndTsFromZeekValues(typ *zng.TypeRecord, tsCol int, vals [][]byte) (zcode.Bytes, nano.Ts, error) {
	if nv, nc := len(vals), len(typ.Columns); nv != nc {
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
		raw = appendZvalFromZeek(raw, typ.Columns[i].Type, val)
	}
	return raw, ts, nil
}

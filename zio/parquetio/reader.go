package parquetio

import (
	"fmt"
	"reflect"

	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/xitongsys/parquet-go/parquet"
	"github.com/xitongsys/parquet-go/reader"
	"github.com/xitongsys/parquet-go/source"
)

type Reader struct {
	pr      *reader.ParquetReader
	typ     *zng.TypeRecord
	columns []parquetColumn
	record  int
}

func NewReader(f source.ParquetFile, zctx *resolver.Context) (*Reader, error) {
	pr, err := reader.NewParquetReader(f, nil, 4)
	if err != nil {
		return nil, err
	}

	cols, err := convertSchema(pr.Footer.Schema)
	if err != nil {
		return nil, err
	}

	zcols := make([]zng.Column, len(cols))
	for i, c := range cols {
		zcols[i] = zng.Column{c.name, c.zngType()}
	}
	typ, err := zctx.LookupTypeRecord(zcols)
	if err != nil {
		return nil, err
	}

	return &Reader{pr, typ, cols, 0}, nil
}

type parquetColumn struct {
	name     string
	ptype    parquet.Type
	ctype    *parquet.ConvertedType
	ltype    *parquet.LogicalType
	listType *parquetColumn
}

func (pc *parquetColumn) zngType() zng.Type {
	if pc.listType != nil {
		inner := pc.listType.zngType()
		return zng.NewTypeArray(-1, inner)
	}

	if pc.ctype != nil {
		switch *pc.ctype {
		case parquet.ConvertedType_UTF8, parquet.ConvertedType_JSON, parquet.ConvertedType_ENUM:
			return zng.TypeString

			// XXX TIMESTAMP_*
			// XXX INTERVAL
			// XXX INT_*, UINT_*
		}
	}

	// XXX handle logical types

	// Unadorned primitive type...
	switch pc.ptype {
	case parquet.Type_BOOLEAN:
		return zng.TypeBool
	case parquet.Type_INT32:
		return zng.TypeInt32
	case parquet.Type_INT64:
		return zng.TypeInt64
	case parquet.Type_FLOAT, parquet.Type_DOUBLE:
		return zng.TypeFloat64
	case parquet.Type_BYTE_ARRAY:
		return zng.TypeBstring

	// XXX
	case parquet.Type_INT96:
		return zng.TypeInt64
	}

	panic(fmt.Sprintf("unhandled parquet type %s", pc.ptype))
}

func (pc *parquetColumn) convert(v reflect.Value, typ zng.Type) (zcode.Bytes, error) {
	if pc.ptype == parquet.Type_INT96 {
		// XXX huh what to do with these
		return zng.EncodeInt(0), nil
	}

	switch typ.ID() {
	case zng.IdBool:
		return zng.EncodeBool(v.Bool()), nil
	case zng.IdInt32, zng.IdInt64:
		return zng.EncodeInt(v.Int()), nil
	case zng.IdFloat64:
		return zng.EncodeFloat64(v.Float()), nil
	case zng.IdBstring, zng.IdString:
		return zng.EncodeBstring(v.String()), nil
	default:
		return nil, fmt.Errorf("unexpected type %s", typ)
	}
}

func (pc *parquetColumn) append(builder *zcode.Builder, v reflect.Value, typ zng.Type) error {
	if pc.listType == nil {
		zv, err := pc.convert(v, typ)
		if err != nil {
			return err
		}
		builder.AppendPrimitive(zv)
		return nil
	}

	arrayType := typ.(*zng.TypeArray)

	builder.BeginContainer()

	n := v.Len()
	for i := 0; i < n; i++ {
		zv, err := pc.listType.convert(v.Index(i), arrayType.Type)
		if err != nil {
			return err
		}
		builder.AppendPrimitive(zv)
	}

	builder.EndContainer()
	return nil
}

func dump(el parquet.SchemaElement) {
	fmt.Printf("%s %s", el.Name, *el.Type)
	if el.ConvertedType != nil {
		fmt.Printf(" ct %s", *el.ConvertedType)
	}
	if el.LogicalType != nil {
		fmt.Printf(" lt %s", *el.LogicalType)
	}
	fmt.Printf("\n")
}

func convertSchema(schema []*parquet.SchemaElement) ([]parquetColumn, error) {
	// build a zng descriptor from the schema.  first element in the
	// schema is the root, skip over it...
	var columns []parquetColumn
	for i := 1; i < len(schema); {
		// dump(*schema[i])

		n := 1
		var col *parquetColumn
		var err error
		if schema[i].NumChildren != nil {
			n, col, err = convertNestedElement(schema, i)
		} else {
			col, err = convertSimpleElement(*schema[i])
		}
		i += n
		if err != nil {
			return nil, err
		}

		// XXX if no error but no type, just skip...
		if col == nil {
			continue
		}

		columns = append(columns, *col)
	}

	return columns, nil
}

func convertSimpleElement(el parquet.SchemaElement) (*parquetColumn, error) {
	if el.RepetitionType != nil && *el.RepetitionType == parquet.FieldRepetitionType_REPEATED {
		return nil, fmt.Errorf("cannot convert repeated element %s", el.Name)
	}

	c := &parquetColumn{el.Name, *el.Type, el.ConvertedType, el.LogicalType, nil}
	return c, nil
}

func convertNestedElement(els []*parquet.SchemaElement, i int) (int, *parquetColumn, error) {
	el := els[i]
	if el.ConvertedType != nil && *el.ConvertedType == parquet.ConvertedType_LIST {
		return convertListType(els, i)
	}
	if el.LogicalType != nil && el.LogicalType.LIST != nil {
		return convertListType(els, i)
	}

	return 1, nil, fmt.Errorf("Cannot handle non-LIST nested element %s", el.Name)
}

func convertListType(els []*parquet.SchemaElement, i int) (int, *parquetColumn, error) {
	// Per https://github.com/apache/parquet-format/blob/master/LogicalTypes.md#lists
	// List structure is:
	// <list-repetition> group <name> (LIST) {
	//   repeated group list {
	//     <element-repetition> <element-type> element;
	//   }
	// }

	el := els[i]
	if len(els) < i+2 {
		return 1, nil, fmt.Errorf("not enough nested elements for LIST %s", el.Name)
	}

	if el.RepetitionType == nil || *el.RepetitionType == parquet.FieldRepetitionType_REPEATED {
		return 1, nil, fmt.Errorf("list (field %s) must not be repeated", el.Name)
	}
	if el.NumChildren == nil || *el.NumChildren != 1 {
		return 1, nil, fmt.Errorf("LIST element (%s) should have 1 child", el.Name)
	}

	listEl := els[i+1]
	if listEl.RepetitionType == nil || *listEl.RepetitionType != parquet.FieldRepetitionType_REPEATED {
		return 1, nil, fmt.Errorf("list (field %s) must not be repeated", el.Name)
	}
	if listEl.NumChildren == nil || *listEl.NumChildren != 1 {
		return 1, nil, fmt.Errorf("LIST element (%s) should have 1 child", el.Name)
	}

	typeEl := els[i+2]
	typ, err := convertSimpleElement(*typeEl)
	if err != nil {
		return 1, nil, err
	}

	c := &parquetColumn{el.Name, *el.Type, el.ConvertedType, el.LogicalType, typ}
	return 3, c, nil
}

func (r *Reader) Read() (*zng.Record, error) {
	if r.record == int(r.pr.GetNumRows()) {
		return nil, nil
	}
	r.record++

	res, err := r.pr.ReadByNumber(1)
	if err != nil {
		return nil, err
	}

	builder := zcode.NewBuilder()
	v := reflect.ValueOf(res[0])
	for i, c := range r.columns {
		fv := v.FieldByName(c.name)
		// parquet-go uses a native type for a required element
		// or a pointer for an optional element.  We should keep
		// track of this when converting the schema, but for now
		// just assume a pointer means optional and add the null
		// value if appropriate or just dereference it...
		if fv.Kind() == reflect.Ptr {
			if fv.IsNil() {
				builder.AppendPrimitive(nil)
				continue
			}
			fv = reflect.Indirect(fv)
		}

		// XXX get rid of the 3rd arg
		err = c.append(builder, fv, r.typ.Columns[i].Type)
		if err != nil {
			return nil, err
		}
	}
	return zng.NewRecord(r.typ, builder.Bytes())
}

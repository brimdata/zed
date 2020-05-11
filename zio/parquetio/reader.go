package parquetio

import (
	"fmt"
	"reflect"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/xitongsys/parquet-go/parquet"
	"github.com/xitongsys/parquet-go/reader"
	"github.com/xitongsys/parquet-go/schema"
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

	cols, err := convertSchema(pr.Footer.Schema, pr.SchemaHandler)
	if err != nil {
		return nil, err
	}

	zcols := make([]zng.Column, len(cols))
	for i, c := range cols {
		zcols[i] = zng.Column{c.name, c.zngType(zctx)}
	}
	typ, err := zctx.LookupTypeRecord(zcols)
	if err != nil {
		return nil, err
	}

	return &Reader{pr, typ, cols, 0}, nil
}

type HandledType int

// These are all the types we can handle...
const (
	// un-annotated primitive types
	Boolean = iota
	Int32
	Int64
	Float
	ByteArray

	// XXX
	Int96

	// annotated strings
	UTF8
	ENUM
	JSON
	BSON

	// annotated int64s
	TimestampMilliseconds
	TimestampMicroseconds
	TimestampNanoseconds

	// XXX INTERVAL
	// XXX INT_*, UINT_* types

	// composite types
	List
)

type parquetColumn struct {
	goName   string
	typ      HandledType
	listType *parquetColumn
	name     string
}

func (pc *parquetColumn) zngType(zctx *resolver.Context) zng.Type {
	if pc.listType != nil {
		inner := pc.listType.zngType(zctx)
		atype := zng.NewTypeArray(-1, inner)
		return zctx.AddType(atype)
	}

	switch pc.typ {
	case Boolean:
		return zng.TypeBool
	case Int32:
		return zng.TypeInt32
	case Int64:
		return zng.TypeInt64
	case Float:
		return zng.TypeFloat64
	case ByteArray:
		return zng.TypeBstring

	case UTF8, ENUM, JSON:
		return zng.TypeString
	case BSON:
		return zng.TypeBstring

	case TimestampMilliseconds, TimestampMicroseconds, TimestampNanoseconds:
		return zng.TypeTime

	// XXX
	case Int96:
		return zng.TypeInt64
	}
	panic(fmt.Sprintf("unhandled type %d", pc.typ))
}

func (pc *parquetColumn) convert(v reflect.Value) (zcode.Bytes, error) {
	switch pc.typ {
	case Int96:
		// XXX huh what to do with these
		return zng.EncodeInt(0), nil

	case Boolean:
		return zng.EncodeBool(v.Bool()), nil

	case Int32, Int64:
		return zng.EncodeInt(v.Int()), nil

	case Float:
		return zng.EncodeFloat64(v.Float()), nil

	case ByteArray, BSON:
		return zng.EncodeBstring(v.String()), nil

	case UTF8, ENUM, JSON:
		return zng.EncodeString(v.String()), nil

	case TimestampMilliseconds:
		return zng.EncodeTime(nano.Ts(v.Int() * 1_000_000)), nil

	case TimestampMicroseconds:
		return zng.EncodeTime(nano.Ts(v.Int() * 1000)), nil

	default:
		return nil, fmt.Errorf("unexpected type %d", pc.typ)
	}
}

func (pc *parquetColumn) append(builder *zcode.Builder, v reflect.Value) error {
	if pc.listType == nil {
		zv, err := pc.convert(v)
		if err != nil {
			return err
		}
		builder.AppendPrimitive(zv)
		return nil
	}

	builder.BeginContainer()

	n := v.Len()
	for i := 0; i < n; i++ {
		zv, err := pc.listType.convert(v.Index(i))
		if err != nil {
			return err
		}
		builder.AppendPrimitive(zv)
	}

	builder.EndContainer()
	return nil
}

func dump(el parquet.SchemaElement) {
	fmt.Printf("%s", el.Name)
	if el.Type == nil {
		fmt.Printf(" (no type)")
	} else {
		fmt.Printf(" %s", *el.Type)
	}

	if el.ConvertedType != nil {
		fmt.Printf(" ct %s", *el.ConvertedType)
	}
	if el.LogicalType != nil {
		fmt.Printf(" lt %s", *el.LogicalType)
	}
	fmt.Printf("\n")
}

func convertSchema(schema []*parquet.SchemaElement, handler *schema.SchemaHandler) ([]parquetColumn, error) {
	rootIn := handler.Infos[0].InName
	rootEx := handler.Infos[0].ExName

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

		// XXX translate the column name
		name := handler.InPathToExPath[fmt.Sprintf("%s.%s", rootIn, col.goName)]
		col.name = name[len(rootEx)+1:]

		columns = append(columns, *col)
	}

	return columns, nil
}

func convertSimpleElement(el parquet.SchemaElement) (*parquetColumn, error) {
	if el.RepetitionType != nil && *el.RepetitionType == parquet.FieldRepetitionType_REPEATED {
		return nil, fmt.Errorf("cannot convert repeated element %s", el.Name)
	}

	var typ HandledType
	if el.ConvertedType != nil {
		switch *el.ConvertedType {
		case parquet.ConvertedType_UTF8:
			typ = UTF8
		case parquet.ConvertedType_JSON:
			typ = JSON
		case parquet.ConvertedType_BSON:
			typ = BSON
		case parquet.ConvertedType_ENUM:
			typ = ENUM
		case parquet.ConvertedType_TIMESTAMP_MILLIS:
			typ = TimestampMilliseconds
		case parquet.ConvertedType_TIMESTAMP_MICROS:
			typ = TimestampMicroseconds

		// XXX case parquet.ConvertedType_INTERVAL:

		default:
			return nil, fmt.Errorf("unhandled ConvertedType %s", *el.ConvertedType)
		}
		// XXX handle logical types
	} else if el.Type != nil {
		switch *el.Type {
		case parquet.Type_BOOLEAN:
			typ = Boolean
		case parquet.Type_INT32:
			typ = Int32
		case parquet.Type_INT64:
			typ = Int64
		case parquet.Type_FLOAT, parquet.Type_DOUBLE:
			typ = Float
		case parquet.Type_BYTE_ARRAY:
			typ = ByteArray
		case parquet.Type_INT96:
			typ = Int96
		default:
			return nil, fmt.Errorf("unhandled type %s\n", *el.Type)
		}
	} else {
		return nil, fmt.Errorf("cannot find type info for %s", el.Name)
	}

	c := &parquetColumn{goName: el.Name, typ: typ}
	return c, nil
}

func countChildren(els []*parquet.SchemaElement, i int) int {
	if i >= len(els) {
		return -1
	}
	if els[i].NumChildren == nil {
		return 1
	}

	n := int(*(els[i].NumChildren))
	j := i + 1
	for c := 0; c < n; c++ {
		cc := countChildren(els, j)
		if cc == -1 {
			return -1
		}
		j += cc
	}
	return j - i
}

func convertNestedElement(els []*parquet.SchemaElement, i int) (int, *parquetColumn, error) {
	el := els[i]
	if el.ConvertedType != nil && *el.ConvertedType == parquet.ConvertedType_LIST {
		return convertListType(els, i)
	}
	if el.LogicalType != nil && el.LogicalType.LIST != nil {
		return convertListType(els, i)
	}

	return countChildren(els, i), nil, nil
	// return 1, nil, fmt.Errorf("Cannot handle non-LIST nested element %s", el.Name)
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

	c := &parquetColumn{goName: el.Name, typ: List, listType: typ}
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
	for _, c := range r.columns {
		fv := v.FieldByName(c.goName)
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

		err = c.append(builder, fv)
		if err != nil {
			return nil, err
		}
	}
	return zng.NewRecord(r.typ, builder.Bytes())
}

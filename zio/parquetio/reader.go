package parquetio

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"

	"github.com/xitongsys/parquet-go/common"
	"github.com/xitongsys/parquet-go/parquet"
	"github.com/xitongsys/parquet-go/reader"
	"github.com/xitongsys/parquet-go/source"
)

const bufsize = 1000

type HandledType int

// These are all the types we can handle...
const (
	// un-annotated primitive types
	boolean = iota
	tint32
	tint64
	float
	double
	byteArray

	// XXX
	int96

	// annotated strings
	utf8
	enum
	json
	bson

	// annotated int64s
	timestampMilliseconds
	timestampMicroseconds
	timestampNanoseconds

	// XXX INTERVAL
	// XXX INT_*, UINT_* types

	// composite types
	list
)

func lookupPrimitiveType(typ *parquet.Type, cType *parquet.ConvertedType) (HandledType, bool) {
	if cType != nil {
		switch *cType {
		case parquet.ConvertedType_UTF8:
			return utf8, true
		case parquet.ConvertedType_JSON:
			return json, true
		case parquet.ConvertedType_BSON:
			return bson, true
		case parquet.ConvertedType_ENUM:
			return enum, true
		case parquet.ConvertedType_TIMESTAMP_MILLIS:
			return timestampMilliseconds, true
		case parquet.ConvertedType_TIMESTAMP_MICROS:
			return timestampMicroseconds, true

		// XXX case parquet.ConvertedType_INTERVAL:

		default:
			return -1, false
		}

		// XXX handle logical types
	} else if typ != nil {
		switch *typ {
		case parquet.Type_BOOLEAN:
			return boolean, true
		case parquet.Type_INT32:
			return tint32, true
		case parquet.Type_INT64:
			return tint64, true
		case parquet.Type_FLOAT:
			return float, true
		case parquet.Type_DOUBLE:
			return double, true
		case parquet.Type_BYTE_ARRAY:
			return byteArray, true
		case parquet.Type_INT96:
			return int96, true
		default:
			return -1, false
		}
	} else {
		return -1, false
	}
}

func simpleParquetTypeToZngType(typ HandledType) zng.Type {
	switch typ {
	case boolean:
		return zng.TypeBool
	case tint32:
		return zng.TypeInt32
	case tint64:
		return zng.TypeInt64
	case float, double:
		return zng.TypeFloat64
	case byteArray:
		return zng.TypeBstring

	case utf8, enum, json:
		return zng.TypeString
	case bson:
		return zng.TypeBstring

	case timestampMilliseconds, timestampMicroseconds, timestampNanoseconds:
		return zng.TypeTime

	// XXX
	case int96:
		return zng.TypeInt64
	}
	panic(fmt.Sprintf("unhandled type %d", typ))
}

func encodeZng(v interface{}, typ HandledType) zcode.Bytes {
	switch typ {
	case boolean:
		return zng.EncodeBool(v.(bool))

	case tint32, tint64:
		return zng.EncodeInt(v.(int64))

	case float, double:
		return zng.EncodeFloat64(v.(float64))

	case byteArray, bson:
		return zng.EncodeBstring(v.(string))

	case utf8, enum, json:
		return zng.EncodeString(v.(string))

	case timestampMilliseconds:
		return zng.EncodeTime(nano.Ts(v.(int) * 1_000_000))

	case timestampMicroseconds:
		return zng.EncodeTime(nano.Ts(v.(int) * 1000))

	default:
		panic(fmt.Sprintf("unexpected type %d", typ))
	}

}

type Reader struct {
	pr      *reader.ParquetReader
	typ     *zng.TypeRecord
	columns []column
	record  int
	total   int
}

func NewReader(f source.ParquetFile, zctx *resolver.Context) (*Reader, error) {
	pr, err := reader.NewParquetReader(f, nil, 1)
	if err != nil {
		return nil, err
	}

	cols, err := buildColumns(pr)
	if err != nil {
		return nil, err
	}

	zcols := make([]zng.Column, len(cols))
	for i, c := range cols {
		zcols[i] = zng.Column{c.getName(), c.zngType(zctx)}
	}
	typ, err := zctx.LookupTypeRecord(zcols)
	if err != nil {
		return nil, err
	}

	return &Reader{pr, typ, cols, 0, int(pr.GetNumRows())}, nil
}

// column abstracts away the handling of an indvidual column from a
// parquet file.  This interface currently has two concrete
// implementations, one for columns that just hold primitive values
// and one for columns that hold lists.
type column interface {
	zngType(zctx *resolver.Context) zng.Type
	append(builder *zcode.Builder) error
	getName() string
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

func buildColumns(pr *reader.ParquetReader) ([]column, error) {
	schema := pr.Footer.Schema

	// first element in the schema is the root, skip it.
	// for each reamaining column, build a column iterator
	// structure.
	var columns []column
	for i := 1; i < len(schema); {
		// dump(*schema[i])

		n := 1
		var col column
		var err error
		if schema[i].NumChildren != nil {
			n, col, err = newNestedColumn(schema, i, pr)
		} else {
			col, err = newSimpleColumn(*schema[i], pr)
		}
		i += n
		if err != nil {
			return nil, err
		}

		// XXX if no error but no type, just skip...
		if col == nil {
			continue
		}

		columns = append(columns, col)
	}

	return columns, nil
}

func newSimpleColumn(el parquet.SchemaElement, pr *reader.ParquetReader) (column, error) {
	if el.RepetitionType != nil && *el.RepetitionType == parquet.FieldRepetitionType_REPEATED {
		return nil, fmt.Errorf("cannot convert repeated element %s", el.Name)
	}

	pT := el.Type
	cT := el.ConvertedType

	typ, ok := lookupPrimitiveType(pT, cT)
	if !ok {
		return nil, errors.New("cannot convert type")
	}

	handler := pr.SchemaHandler
	path := []string{handler.Infos[0].InName, el.Name}
	pathStr := common.PathToStr(path)

	// The parquet-go library converts column names into names
	// that are valid public field names in a go structure.
	// Recover the original column names from the parquet schema
	// here.  This is a little messy since the data structure
	// inside parquet-go uses fully qualified names so we have
	// to convert a field name to the fully qualified name, map
	// it to the original fully qualified name, then grab the
	// original column name.
	name := handler.InPathToExPath[pathStr]
	name = name[len(handler.Infos[0].ExName)+1:]

	maxRepetition, _ := handler.MaxRepetitionLevel(path)
	maxDefinition, _ := handler.MaxDefinitionLevel(path)

	iter := newColumnIterator(pr, el.Name, maxRepetition, maxDefinition)
	return &simpleColumn{
		name:          name,
		typ:           typ,
		iter:          iter,
		maxDefinition: maxDefinition,
		pT:            pT,
		cT:            cT,
	}, nil
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

func newNestedColumn(els []*parquet.SchemaElement, i int, pr *reader.ParquetReader) (int, column, error) {
	el := els[i]
	if el.ConvertedType != nil && *el.ConvertedType == parquet.ConvertedType_LIST {
		return newListColumn(els, i, pr)
	}
	if el.LogicalType != nil && el.LogicalType.LIST != nil {
		return newListColumn(els, i, pr)
	}

	// Skip this element and all its children...
	return countChildren(els, i), nil, nil
}

func newListColumn(els []*parquet.SchemaElement, i int, pr *reader.ParquetReader) (int, column, error) {
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

	pT := typeEl.Type
	cT := typeEl.ConvertedType

	typ, ok := lookupPrimitiveType(pT, cT)
	if !ok {
		return 3, nil, errors.New("cannot convert type")
	}

	handler := pr.SchemaHandler
	path := []string{handler.Infos[0].InName, el.Name, listEl.Name, typeEl.Name}

	// The parquet-go library converts column names into names
	// that are valid public field names in a go structure.
	// Recover the original column names from the parquet schema
	// here.  This is a little messy since the data structure
	// inside parquet-go uses fully qualified names so we have
	// to convert a field name to the fully qualified name, map
	// it to the original fully qualified name, then grab the
	// original column name.
	name := handler.InPathToExPath[common.PathToStr(path[:2])]
	name = name[len(handler.Infos[0].ExName)+1:]

	maxRepetition, _ := handler.MaxRepetitionLevel(path)
	maxDefinition, _ := handler.MaxDefinitionLevel(path)

	iter := newColumnIterator(pr, el.Name, maxRepetition, maxDefinition)

	c := listColumn{
		name:          name,
		innerType:     typ,
		iter:          iter,
		maxRepetition: maxRepetition,
		maxDefinition: maxDefinition,
	}

	return 3, &c, nil
}

// simpleColumn handles a column from a parquet file that holds individual
// (non-repeated) primitive values.
type simpleColumn struct {
	name string
	typ  HandledType

	iter          *columnIterator
	maxDefinition int32

	pT *parquet.Type
	cT *parquet.ConvertedType
}

func (c *simpleColumn) getName() string { return c.name }

func (c *simpleColumn) zngType(zctx *resolver.Context) zng.Type {
	return simpleParquetTypeToZngType(c.typ)
}

func appendItem(builder *zcode.Builder, typ HandledType, iter *columnIterator, maxDef, maxRep int32) (bool, error) {
	var rl, dl int32
	switch typ {
	case boolean:
		var b bool
		b, rl, dl = iter.nextBoolean()
		if maxDef > dl {
			builder.AppendPrimitive(nil)
		} else {
			builder.AppendPrimitive(zng.EncodeBool(b))
		}
	case tint32:
		var i int32
		i, rl, dl = iter.nextInt32()
		if maxDef > dl {
			builder.AppendPrimitive(nil)
		} else {
			builder.AppendPrimitive(zng.EncodeInt(int64(i)))
		}
	case tint64:
		var i int64
		i, rl, dl = iter.nextInt64()
		if maxDef > dl {
			builder.AppendPrimitive(nil)
		} else {
			builder.AppendPrimitive(zng.EncodeInt(i))
		}
	case float:
		var f float64
		f, rl, dl = iter.nextFloat()
		if maxDef > dl {
			builder.AppendPrimitive(nil)
		} else {
			builder.AppendPrimitive(zng.EncodeFloat64(f))
		}
	case double:
		var f float64
		f, rl, dl = iter.nextDouble()
		if maxDef > dl {
			builder.AppendPrimitive(nil)
		} else {
			builder.AppendPrimitive(zng.EncodeFloat64(f))
		}
	case utf8, enum, json:
		var a []byte
		a, rl, dl = iter.nextByteArray()
		if maxDef > dl {
			builder.AppendPrimitive(nil)
		} else {
			builder.AppendPrimitive(zng.EncodeString(string(a)))
		}
	case byteArray, bson:
		var a []byte
		a, rl, dl = iter.nextByteArray()
		if maxDef > dl {
			builder.AppendPrimitive(nil)
		} else {
			builder.AppendPrimitive(zng.EncodeBstring(string(a)))
		}
		//case timestampMilliseconds, timestampMicroseconds, timestampNanoseconds:
		// XXX
	default:
		return false, fmt.Errorf("unhandled type %d", typ)
	}
	return (rl == maxRep), nil
}

// append reads the next value from this column and appends it to the
// given zcode.Builder.  This code represents an unwound and vastly
// simplified version of the code in the methods:
// parquet-go.reader.ParquetReader.read(), and
// parquet-go.marshal.Unmarshal()
func (c *simpleColumn) append(builder *zcode.Builder) error {
	_, err := appendItem(builder, c.typ, c.iter, c.maxDefinition, 0)
	return err
}

// listColumn handles a column from a parquet file that holds LIST
// structures as defined in the parquet spec.
type listColumn struct {
	name      string
	innerType HandledType

	iter          *columnIterator
	maxRepetition int32
	maxDefinition int32
}

func (c *listColumn) getName() string { return c.name }

func (c *listColumn) zngType(zctx *resolver.Context) zng.Type {
	inner := simpleParquetTypeToZngType(c.innerType)
	return zctx.LookupTypeArray(inner)
}

// append reads the next value from this column and appends it to the given
// zcode.Builder.  This code (together with the readNext() method represent
// an unwound and vastly simplified version of the code in the methods:
// parquet-go.reader.ParquetReader.read(), and
// parquet-go.marshal.Unmarshal()
func (c *listColumn) append(builder *zcode.Builder) error {
	dl := c.iter.peekDL()
	if c.maxDefinition > dl {
		builder.AppendContainer(nil)
		return nil
	}

	builder.BeginContainer()
	for {
		last, err := appendItem(builder, c.innerType, c.iter, c.maxDefinition, c.maxRepetition)
		if err != nil {
			return err
		}
		if last {
			break
		}
	}
	builder.EndContainer()
	return nil
}

func (r *Reader) Read() (*zng.Record, error) {
	if r.record == r.total {
		return nil, nil
	}
	r.record++

	builder := zcode.NewBuilder()
	for _, c := range r.columns {
		err := c.append(builder)
		if err != nil {
			return nil, err
		}
	}
	return zng.NewRecord(r.typ, builder.Bytes())
}

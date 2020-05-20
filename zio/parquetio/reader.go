package parquetio

import (
	"errors"
	"fmt"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
	"github.com/xitongsys/parquet-go/common"
	"github.com/xitongsys/parquet-go/layout"
	"github.com/xitongsys/parquet-go/parquet"
	"github.com/xitongsys/parquet-go/reader"
	"github.com/xitongsys/parquet-go/source"
	"github.com/xitongsys/parquet-go/types"
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
			return HandledType(-1), false
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
		case parquet.Type_FLOAT, parquet.Type_DOUBLE:
			return float, true
		case parquet.Type_BYTE_ARRAY:
			return byteArray, true
		case parquet.Type_INT96:
			return int96, true
		default:
			return HandledType(-1), false
		}
	} else {
		return HandledType(-1), false
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
	case float:
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

	case float:
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

	cbuf := pr.ColumnBuffers[pathStr]

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

	// maxRepetition, _ := schemaHandler.MaxRepetitionLevel(path)
	maxDefinition, _ := handler.MaxDefinitionLevel(path)

	return &simpleColumn{
		name: name,
		typ:  typ,
		cbuf: cbuf,
		pT:   pT,
		cT:   cT,
		//maxRepetition: repetitionLevels,
		maxDefinition: maxDefinition,
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

	cbuf := pr.ColumnBuffers[common.PathToStr(path)]

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

	c := listColumn{
		name:          name,
		innerType:     typ,
		cbuf:          cbuf,
		pT:            pT,
		cT:            cT,
		maxRepetition: maxRepetition,
		maxDefinition: maxDefinition,
	}

	return 3, &c, nil
}

type simpleColumn struct {
	name      string
	typ       HandledType
	cbuf      *reader.ColumnBufferType
	table     *layout.Table
	n         int
	tableSize int

	pT            *parquet.Type
	cT            *parquet.ConvertedType
	maxDefinition int32
}

func (c *simpleColumn) getName() string { return c.name }

func (c *simpleColumn) zngType(zctx *resolver.Context) zng.Type {
	return simpleParquetTypeToZngType(c.typ)
}

func (c *simpleColumn) append(builder *zcode.Builder) error {
	if c.n >= c.tableSize {
		tab, nread := c.cbuf.ReadRows(int64(bufsize))
		c.table = tab
		c.tableSize = int(nread)
		c.n = 0
	}

	dl := c.table.DefinitionLevels[c.n]
	val := c.table.Values[c.n]
	c.n++

	if c.maxDefinition > dl {
		builder.AppendPrimitive(nil)
		return nil
	}

	v := types.ParquetTypeToGoType(val, c.pT, c.cT)
	builder.AppendPrimitive(encodeZng(v, c.typ))
	return nil
}

type listColumn struct {
	name      string
	innerType HandledType

	cbuf      *reader.ColumnBufferType
	table     *layout.Table
	n         int
	tableSize int

	pT            *parquet.Type
	cT            *parquet.ConvertedType
	maxRepetition int32
	maxDefinition int32
}

func (c *listColumn) getName() string { return c.name }

func (c *listColumn) zngType(zctx *resolver.Context) zng.Type {
	inner := simpleParquetTypeToZngType(c.innerType)
	return zctx.LookupTypeArray(inner)
}

func (c *listColumn) readNext() (interface{}, int32, int32) {
	if c.n >= c.tableSize {
		tab, _ := c.cbuf.ReadRows(int64(bufsize))
		c.table = tab
		ln := len(tab.RepetitionLevels)
		if ln != len(tab.DefinitionLevels) {
			panic("mismatched repetition/definition levels")
		}
		if ln != len(tab.Values) {
			panic("mismatched repetition levels/values")
		}
		c.tableSize = ln
		c.n = 0
	}

	rl := c.table.RepetitionLevels[c.n]
	dl := c.table.DefinitionLevels[c.n]
	val := c.table.Values[c.n]
	c.n++

	return val, rl, dl
}

func (c *listColumn) append(builder *zcode.Builder) error {
	v, rl, dl := c.readNext()

	if c.maxDefinition > dl {
		builder.AppendPrimitive(nil)
		return nil
	}

	builder.BeginContainer()
	for {
		if v == nil {
			builder.AppendPrimitive(nil)
		} else {
			builder.AppendPrimitive(encodeZng(v, c.innerType))
		}
		if rl == c.maxRepetition {
			break
		}
		v, rl, dl = c.readNext()
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

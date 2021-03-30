package parquetio

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zng/resolver"
	"github.com/fraugster/parquet-go/parquet"
	"github.com/fraugster/parquet-go/parquetschema"
)

func newRecordType(zctx *resolver.Context, children []*parquetschema.ColumnDefinition) (*zng.TypeRecord, error) {
	var cols []zng.Column
	for _, c := range children {
		typ, err := newType(zctx, c)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", c.SchemaElement.Name, err)
		}
		cols = append(cols, zng.Column{
			Name: c.SchemaElement.Name,
			Type: typ,
		})
	}
	return zctx.LookupTypeRecord(cols)
}

func newType(zctx *resolver.Context, cd *parquetschema.ColumnDefinition) (zng.Type, error) {
	se := cd.SchemaElement
	if se.Type != nil {
		return newPrimitiveType(zctx, se)
	}
	if se.ConvertedType != nil {
		switch *se.ConvertedType {
		case parquet.ConvertedType_MAP:
			keyType, err := newType(zctx, cd.Children[0].Children[0])
			if err != nil {
				return nil, fmt.Errorf("%s: map key: %w", cd.SchemaElement.Name, err)
			}
			valType, err := newType(zctx, cd.Children[0].Children[1])
			if err != nil {
				return nil, fmt.Errorf("%s: map value: %w", cd.SchemaElement.Name, err)
			}
			return zctx.LookupTypeMap(keyType, valType), nil

		case parquet.ConvertedType_LIST:
			typ, err := newType(zctx, cd.Children[0].Children[0])
			if err != nil {
				return nil, err
			}
			return zctx.LookupTypeArray(typ), nil
		}
	}
	return newRecordType(zctx, cd.Children)

}

func newPrimitiveType(zctx *resolver.Context, s *parquet.SchemaElement) (zng.Type, error) {
	if s.IsSetLogicalType() && s.LogicalType.IsSetDECIMAL() ||
		s.GetConvertedType() == parquet.ConvertedType_DECIMAL {
		return nil, errors.New("DECIMAL type is unimplemented")
	}
	switch *s.Type {
	case parquet.Type_BOOLEAN:
		return zng.TypeBool, nil
	case parquet.Type_INT32:
		if s.IsSetLogicalType() {
			switch l := s.LogicalType; {
			case l.IsSetDATE():
				zctx.LookupTypeAlias("date", zng.TypeInt32)
			case l.IsSetINTEGER():
				switch i := l.INTEGER; {
				case i.BitWidth == 8 && i.IsSigned:
					return zng.TypeInt8, nil
				case i.BitWidth == 8:
					return zng.TypeUint8, nil
				case i.BitWidth == 16 && i.IsSigned:
					return zng.TypeInt16, nil
				case i.BitWidth == 16:
					return zng.TypeUint16, nil
				case i.BitWidth == 32 && i.IsSigned:
					return zng.TypeInt32, nil
				case i.BitWidth == 32:
					return zng.TypeUint32, nil
				}
			case l.IsSetTIME() && l.TIME.IsSetUnit() && l.TIME.Unit.IsSetMILLIS():
				return zctx.LookupTypeAlias("time_millis", zng.TypeInt32)
			}
		}
		if s.IsSetConvertedType() {
			switch *s.ConvertedType {
			case parquet.ConvertedType_DATE:
				return zctx.LookupTypeAlias("date", zng.TypeInt32)
			case parquet.ConvertedType_UINT_8:
				return zng.TypeUint8, nil
			case parquet.ConvertedType_UINT_16:
				return zng.TypeUint16, nil
			case parquet.ConvertedType_UINT_32:
				return zng.TypeUint32, nil
			case parquet.ConvertedType_INT_8:
				return zng.TypeInt8, nil
			case parquet.ConvertedType_INT_16:
				return zng.TypeInt16, nil
			case parquet.ConvertedType_INT_32:
				return zng.TypeInt32, nil
			case parquet.ConvertedType_TIME_MILLIS:
				return zctx.LookupTypeAlias("time_millis", zng.TypeInt32)
			}
		}
		return zng.TypeInt32, nil
	case parquet.Type_INT64:
		if s.IsSetLogicalType() {
			switch l := s.LogicalType; {
			case l.IsSetINTEGER():
				switch {
				case l.INTEGER.BitWidth == 64 && l.INTEGER.IsSigned:
					return zng.TypeInt64, nil
				case l.INTEGER.BitWidth == 64:
					return zng.TypeUint64, nil
				}
			case l.IsSetTIME() && l.TIME.IsSetUnit():
				switch {
				case l.TIME.Unit.IsSetMICROS():
					return zctx.LookupTypeAlias("time_micros", zng.TypeInt64)
				case l.TIME.Unit.IsSetNANOS():
					return zctx.LookupTypeAlias("time_nanos", zng.TypeInt64)
				}
			case l.IsSetTIMESTAMP() && l.TIMESTAMP.IsSetUnit():
				switch {
				case l.TIMESTAMP.Unit.IsSetMILLIS():
					return zctx.LookupTypeAlias("timestamp_millis", zng.TypeInt64)
				case l.TIMESTAMP.Unit.IsSetMICROS():
					return zctx.LookupTypeAlias("timestamp_micros", zng.TypeInt64)
				case l.TIMESTAMP.Unit.IsSetNANOS():
					return zng.TypeTime, nil
				}
			}
		}
		if s.IsSetConvertedType() {
			switch *s.ConvertedType {
			case parquet.ConvertedType_UINT_64:
				return zng.TypeUint64, nil
			case parquet.ConvertedType_INT_64:
				return zng.TypeInt64, nil
			case parquet.ConvertedType_TIME_MICROS:
				return zctx.LookupTypeAlias("time_micros", zng.TypeInt64)
			case parquet.ConvertedType_TIMESTAMP_MILLIS:
				return zctx.LookupTypeAlias("timestamp_millis", zng.TypeInt32)
			case parquet.ConvertedType_TIMESTAMP_MICROS:
				return zctx.LookupTypeAlias("timestamp_micros", zng.TypeInt64)
			}
		}
		return zng.TypeInt64, nil
	case parquet.Type_INT96:
		return zctx.LookupTypeAlias("int96", zng.TypeBytes)
	case parquet.Type_FLOAT:
		return zctx.LookupTypeAlias("float", zng.TypeFloat64)
	case parquet.Type_DOUBLE:
		return zng.TypeFloat64, nil
	case parquet.Type_BYTE_ARRAY:
		if s.IsSetLogicalType() {
			switch l := s.LogicalType; {
			case l.IsSetBSON():
				return zctx.LookupTypeAlias("bson", zng.TypeBytes)
			case l.IsSetENUM():
				return zctx.LookupTypeAlias("enum", zng.TypeString)
			case l.IsSetJSON():
				return zctx.LookupTypeAlias("json", zng.TypeString)
			case l.IsSetSTRING():
				return zng.TypeString, nil
			}
		}
		if s.IsSetConvertedType() {
			switch *s.ConvertedType {
			case parquet.ConvertedType_BSON:
				return zctx.LookupTypeAlias("bson", zng.TypeBytes)
			case parquet.ConvertedType_JSON:
				return zctx.LookupTypeAlias("json", zng.TypeString)
			case parquet.ConvertedType_ENUM:
				return zctx.LookupTypeAlias("enum", zng.TypeString)
			case parquet.ConvertedType_UTF8:
				return zng.TypeString, nil
			}
		}
		return zng.TypeBytes, nil
	case parquet.Type_FIXED_LEN_BYTE_ARRAY:
		switch {
		case s.GetTypeLength() == 16 && s.IsSetLogicalType() && s.LogicalType.IsSetUUID():
			return zctx.LookupTypeAlias("uuid", zng.TypeBytes)
		case s.GetTypeLength() == 12 && s.GetConvertedType() == parquet.ConvertedType_INTERVAL:
			return zctx.LookupTypeAlias("interval", zng.TypeBytes)
		}
		return zctx.LookupTypeAlias(fmt.Sprintf("fixed_len_byte_array_%d", *s.TypeLength), zng.TypeBytes)
	}
	panic(s.Type.String())
}

package parquetio

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/fraugster/parquet-go/parquet"
	"github.com/fraugster/parquet-go/parquetschema"
)

func newRecordType(zctx *zed.Context, children []*parquetschema.ColumnDefinition) (*zed.TypeRecord, error) {
	var cols []zed.Column
	for _, c := range children {
		typ, err := newType(zctx, c)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", c.SchemaElement.Name, err)
		}
		cols = append(cols, zed.Column{
			Name: c.SchemaElement.Name,
			Type: typ,
		})
	}
	return zctx.LookupTypeRecord(cols)
}

func newType(zctx *zed.Context, cd *parquetschema.ColumnDefinition) (zed.Type, error) {
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

func newPrimitiveType(zctx *zed.Context, s *parquet.SchemaElement) (zed.Type, error) {
	if s.IsSetLogicalType() && s.LogicalType.IsSetDECIMAL() ||
		s.GetConvertedType() == parquet.ConvertedType_DECIMAL {
		return nil, errors.New("DECIMAL type is unimplemented")
	}
	switch *s.Type {
	case parquet.Type_BOOLEAN:
		return zed.TypeBool, nil
	case parquet.Type_INT32:
		if s.IsSetLogicalType() {
			switch l := s.LogicalType; {
			case l.IsSetDATE():
				zctx.LookupTypeAlias("date", zed.TypeInt32)
			case l.IsSetINTEGER():
				switch i := l.INTEGER; {
				case i.BitWidth == 8 && i.IsSigned:
					return zed.TypeInt8, nil
				case i.BitWidth == 8:
					return zed.TypeUint8, nil
				case i.BitWidth == 16 && i.IsSigned:
					return zed.TypeInt16, nil
				case i.BitWidth == 16:
					return zed.TypeUint16, nil
				case i.BitWidth == 32 && i.IsSigned:
					return zed.TypeInt32, nil
				case i.BitWidth == 32:
					return zed.TypeUint32, nil
				}
			case l.IsSetTIME() && l.TIME.IsSetUnit() && l.TIME.Unit.IsSetMILLIS():
				return zctx.LookupTypeAlias("time_millis", zed.TypeInt32)
			}
		}
		if s.IsSetConvertedType() {
			switch *s.ConvertedType {
			case parquet.ConvertedType_DATE:
				return zctx.LookupTypeAlias("date", zed.TypeInt32)
			case parquet.ConvertedType_UINT_8:
				return zed.TypeUint8, nil
			case parquet.ConvertedType_UINT_16:
				return zed.TypeUint16, nil
			case parquet.ConvertedType_UINT_32:
				return zed.TypeUint32, nil
			case parquet.ConvertedType_INT_8:
				return zed.TypeInt8, nil
			case parquet.ConvertedType_INT_16:
				return zed.TypeInt16, nil
			case parquet.ConvertedType_INT_32:
				return zed.TypeInt32, nil
			case parquet.ConvertedType_TIME_MILLIS:
				return zctx.LookupTypeAlias("time_millis", zed.TypeInt32)
			}
		}
		return zed.TypeInt32, nil
	case parquet.Type_INT64:
		if s.IsSetLogicalType() {
			switch l := s.LogicalType; {
			case l.IsSetINTEGER():
				switch {
				case l.INTEGER.BitWidth == 64 && l.INTEGER.IsSigned:
					return zed.TypeInt64, nil
				case l.INTEGER.BitWidth == 64:
					return zed.TypeUint64, nil
				}
			case l.IsSetTIME() && l.TIME.IsSetUnit():
				switch {
				case l.TIME.Unit.IsSetMICROS():
					return zctx.LookupTypeAlias("time_micros", zed.TypeInt64)
				case l.TIME.Unit.IsSetNANOS():
					return zctx.LookupTypeAlias("time_nanos", zed.TypeInt64)
				}
			case l.IsSetTIMESTAMP() && l.TIMESTAMP.IsSetUnit():
				switch {
				case l.TIMESTAMP.Unit.IsSetMILLIS():
					return zctx.LookupTypeAlias("timestamp_millis", zed.TypeInt64)
				case l.TIMESTAMP.Unit.IsSetMICROS():
					return zctx.LookupTypeAlias("timestamp_micros", zed.TypeInt64)
				case l.TIMESTAMP.Unit.IsSetNANOS():
					return zed.TypeTime, nil
				}
			}
		}
		if s.IsSetConvertedType() {
			switch *s.ConvertedType {
			case parquet.ConvertedType_UINT_64:
				return zed.TypeUint64, nil
			case parquet.ConvertedType_INT_64:
				return zed.TypeInt64, nil
			case parquet.ConvertedType_TIME_MICROS:
				return zctx.LookupTypeAlias("time_micros", zed.TypeInt64)
			case parquet.ConvertedType_TIMESTAMP_MILLIS:
				return zctx.LookupTypeAlias("timestamp_millis", zed.TypeInt32)
			case parquet.ConvertedType_TIMESTAMP_MICROS:
				return zctx.LookupTypeAlias("timestamp_micros", zed.TypeInt64)
			}
		}
		return zed.TypeInt64, nil
	case parquet.Type_INT96:
		return zctx.LookupTypeAlias("int96", zed.TypeBytes)
	case parquet.Type_FLOAT:
		return zed.TypeFloat32, nil
	case parquet.Type_DOUBLE:
		return zed.TypeFloat64, nil
	case parquet.Type_BYTE_ARRAY:
		if s.IsSetLogicalType() {
			switch l := s.LogicalType; {
			case l.IsSetBSON():
				return zctx.LookupTypeAlias("bson", zed.TypeBytes)
			case l.IsSetENUM():
				return zctx.LookupTypeAlias("enum", zed.TypeString)
			case l.IsSetJSON():
				return zctx.LookupTypeAlias("json", zed.TypeString)
			case l.IsSetSTRING():
				return zed.TypeString, nil
			}
		}
		if s.IsSetConvertedType() {
			switch *s.ConvertedType {
			case parquet.ConvertedType_BSON:
				return zctx.LookupTypeAlias("bson", zed.TypeBytes)
			case parquet.ConvertedType_JSON:
				return zctx.LookupTypeAlias("json", zed.TypeString)
			case parquet.ConvertedType_ENUM:
				return zctx.LookupTypeAlias("enum", zed.TypeString)
			case parquet.ConvertedType_UTF8:
				return zed.TypeString, nil
			}
		}
		return zed.TypeBytes, nil
	case parquet.Type_FIXED_LEN_BYTE_ARRAY:
		switch {
		case s.GetTypeLength() == 16 && s.IsSetLogicalType() && s.LogicalType.IsSetUUID():
			return zctx.LookupTypeAlias("uuid", zed.TypeBytes)
		case s.GetTypeLength() == 12 && s.GetConvertedType() == parquet.ConvertedType_INTERVAL:
			return zctx.LookupTypeAlias("interval", zed.TypeBytes)
		}
		return zctx.LookupTypeAlias(fmt.Sprintf("fixed_len_byte_array_%d", *s.TypeLength), zed.TypeBytes)
	}
	panic(s.Type.String())
}

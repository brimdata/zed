package zeekio

import (
	"errors"
	"fmt"

	"github.com/brimdata/zed"
)

var ErrIncompatibleZeekType = errors.New("type cannot be represented in zeek format")

func zngTypeToZeek(typ zed.Type) (string, error) {
	switch typ := typ.(type) {
	case *zed.TypeArray:
		inner, err := zngTypeToZeek(typ.Type)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("vector[%s]", inner), nil
	case *zed.TypeSet:
		inner, err := zngTypeToZeek(typ.Type)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("set[%s]", inner), nil
	case *zed.TypeOfUint8, *zed.TypeOfInt8, *zed.TypeOfInt16, *zed.TypeOfInt32, *zed.TypeOfInt64, *zed.TypeOfUint16, *zed.TypeOfUint32:
		return "int", nil
	case *zed.TypeOfUint64:
		return "count", nil
	case *zed.TypeOfFloat16, *zed.TypeOfFloat32, *zed.TypeOfFloat64:
		return "double", nil
	case *zed.TypeOfIP:
		return "addr", nil
	case *zed.TypeOfNet:
		return "subnet", nil
	case *zed.TypeOfDuration:
		return "interval", nil
	case *zed.TypeNamed:
		if typ.Name == "zenum" {
			return "enum", nil
		}
		if typ.Name == "port" {
			return "port", nil
		}
		return zngTypeToZeek(typ.Type)
	case *zed.TypeOfBool, *zed.TypeOfString, *zed.TypeOfTime:
		return zed.PrimitiveName(typ), nil
	default:
		return "", fmt.Errorf("type %s: %w", typ, ErrIncompatibleZeekType)
	}
}

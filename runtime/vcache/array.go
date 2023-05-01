package vcache

import (
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/pkg/field"
	"github.com/brimdata/zed/vector"
	"github.com/brimdata/zed/vng"
	meta "github.com/brimdata/zed/vng/vector" //XXX rename package
)

func loadArray(any *vector.Any, typ zed.Type, path field.Path, m *meta.Array, r io.ReaderAt) (*vector.Array, error) {
	if *any == nil {
		var innerType zed.Type
		switch typ := typ.(type) {
		case *zed.TypeArray:
			innerType = typ.Type
		case *zed.TypeSet:
			innerType = typ.Type
		default:
			return nil, fmt.Errorf("internal error: vcache.loadArray encountered bad type: %s", typ)
		}
		lengths, err := vng.ReadIntVector(m.Lengths, r)
		if err != nil {
			return nil, err
		}
		values, err := loadVector(nil, innerType, path, m.Values, r)
		if err != nil {
			return nil, err
		}
		*any = vector.NewArray(typ.(*zed.TypeArray), lengths, values)
	}
	//XXX always return the array as the vector engine needs to know how to handle
	// manipulating the array no matter what it contains
	return (*any).(*vector.Array), nil
}

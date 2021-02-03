package agg

import (
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type Shape struct {
	shapes map[*zng.TypeRecord]struct{}
}

func newShape() *Shape {
	return &Shape{
		shapes: make(map[*zng.TypeRecord]struct{}),
	}
}

func (s *Shape) Consume(v zng.Value) error {
	// only works for record types, e.g., shape(foo.x) where foo.x is a record
	typ, ok := v.Type.(*zng.TypeRecord)
	if !ok {
		//XXX bump counter
		return nil
	}
	s.shapes[typ] = struct{}{}
	return nil
}

func (s *Shape) Result(zctx *resolver.Context) (zng.Value, error) {
	if len(s.shapes) == 0 {
		// empty input
		return zng.Value{zng.TypeNull, nil}, nil
	}
	schema, _ := newSchema(zctx)
	for typ := range s.shapes {
		schema.mixin(typ)
	}
	return zng.Value{zng.TypeType, zcode.Bytes(schema.typ.ZSON())}, nil
}

// XXX add TBD_ so spilling doesn't work yet.

func (s *Shape) ConsumeAsPartial(zv zng.Value) error {
	//XXX we need zson parser to turn type value back into record type
	return nil
}

func (s *Shape) ResultAsPartial(*resolver.Context) (zng.Value, error) {
	return s.Result(nil)
}

package reducer

import (
	"github.com/brimsec/zq/expr"
	"github.com/brimsec/zq/proc/fuse"
	"github.com/brimsec/zq/zcode"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type Shape struct {
	Reducer
	zctx   *resolver.Context
	arg    expr.Evaluator
	shapes map[*zng.TypeRecord]struct{}
}

func newShape(zctx *resolver.Context, arg, where expr.Evaluator) *Shape {
	return &Shape{
		Reducer: Reducer{where: where},
		zctx:    zctx,
		arg:     arg,
		shapes:  make(map[*zng.TypeRecord]struct{}),
	}
}

func (s *Shape) Consume(r *zng.Record) {
	if s.filter(r) {
		return
	}
	v, err := s.arg.Eval(r)
	if err != nil {
		return
	}
	// only works for record types, e.g., shape(foo.x) where foo.x is a record
	typ, ok := v.Type.(*zng.TypeRecord)
	if !ok {
		//XXX bump counter
		return
	}
	s.shapes[typ] = struct{}{}
}

func (s *Shape) Result() zng.Value {
	if len(s.shapes) == 0 {
		// empty input
		return zng.Value{zng.TypeNull, nil}
	}
	shaper := fuse.NewSchema()
	for typ := range s.shapes {
		shaper.Mixin(typ)
	}
	shape, err := s.zctx.LookupTypeRecord(shaper.Columns())
	if err != nil {
		//XXX
		return zng.NewErrorf("internal error")
	}
	return zng.Value{zng.TypeType, zcode.Bytes(shape.ZSON())}
}

// XXX add TBD_ so spilling doesn't work yet.

func (s *Shape) TBD_ConsumePart(zv zng.Value) error {
	//XXX we need zson parser to turn type value back into record type
	return nil
}

func (s *Shape) TBD_ResultPart(*resolver.Context) (zng.Value, error) {
	return s.Result(), nil
}

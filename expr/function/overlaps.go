package function

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zson"
	"github.com/brimdata/zed/expr/extent"
	"github.com/brimdata/zed/order"
)

// https://github.com/brimdata/zed/blob/main/docs/language/functions.md#overlaps
type Overlaps struct {
	zctx *zed.Context
}

type segment struct {
	from     *zed.Value
	to       *zed.Value
	record *zed.Value
}

func (o *Overlaps) Call(ectx zed.Allocator, args []zed.Value) *zed.Value {
	// array of record with from/to keys
	arrayType := zed.TypeUnder(args[0].Type)
	if _, ok := arrayType.(*zed.TypeArray); !ok {
		return newErrorf(o.zctx, ectx, "overlaps: %q not an array of from/to records", zson.String(args[0]))
	}
	recType := zed.TypeRecordOf(arrayType)
	if recType == nil {
		return newErrorf(o.zctx, ectx, "overlaps: %q not an array of from/to records", zson.String(args[0]))
	}
        var segments []segment
	it := args[0].Iter()
	for !it.Done() {
                val := zed.NewValue(recType, it.Next())
                from := val.Deref("from")
                if from.IsMissing() {
                        return newErrorf(o.zctx, ectx, "overlaps: item missing 'from' field: %s", val)
                }
                to := val.Deref("to")
                if from.IsMissing() {
                        return newErrorf(o.zctx, ectx, "overlaps: item missing 'to' field: %s", val)
                }
                segments = append(segments, segment{
                        from: from,
                        to: to,
                        record: zed.NewValue(recType, it.Next()),
                })
	}
        ord := order.Asc //XXX parameter
        cmp := extent.CompareFunc(ord)
	spans := sortedObjectSpans(objects, cmp)
	var s stack
	s.pushObjectSpan(spans[0], cmp)
	for _, span := range spans[1:] {
		tos := s.tos()
		if span.Before(tos.Last()) {
			s.pushObjectSpan(span, cmp)
		} else {
			tos.Objects = append(tos.Objects, data.NewObjectScan(*span.object))
			tos.Extend(span.Last())
		}
	}
	// On exit, the ranges in the stack are properly sorted so
	// we just return the stack as a []Range.
	return s

}

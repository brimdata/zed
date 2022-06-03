package seekindex

import (
	"context"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/compiler"
	"github.com/brimdata/zed/compiler/ast/dag"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/runtime/expr/extent"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zfmt"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
)

func Lookup(ctx context.Context, reader zio.Reader, keyfilter *dag.KeyFilter, index extent.Span, lastKey *zed.Value, count uint64, size int64, o order.Which) (Range, error) {
	rg := Range{End: size}
	src := `put num := count() - 1
| yield {...this, bucket: num - num %% 2}, {...this, bucket: num + num %% 2 - 1}
| keys := union(key), counts := union(count), offsets := union(offset) by bucket
| drop bucket
| where len(keys) == 2
| over flatten(this) => (
    over value with key => (
      sort this
      | collect(this)
      | {key,value:{lower:collect[0],upper:collect[1]}}
    )
    | collect(this)
)
| unflatten(collect)
| {...offsets, ...this}
| drop offsets
| sort lower
| where %s
| where %s
| fork ( => head 1 => tail 1 )`
	poolKeyExpr := "true"
	if keyfilter != nil {
		poolKeyExpr = zfmt.DAGExpr(keyfilter.SpanFilter(o, "keys"))
	}
	indexExpr := "true"
	if index != nil {
		indexExpr = fmt.Sprintf("counts.upper >= %s and counts.lower <= %s",
			zson.MustFormatValue(index.First()),
			zson.MustFormatValue(index.Last()),
		)
	}
	src = fmt.Sprintf(src, poolKeyExpr, indexExpr)
	program, err := compiler.ParseOp(src)
	if err != nil {
		return rg, err
	}
	lastval, err := zson.MarshalZNG(Entry{
		Key:    lastKey,
		Offset: size,
		Count:  count,
	})
	if err != nil {
		return rg, err
	}
	reader = zio.ConcatReader(reader, zbuf.NewArray([]zed.Value{*lastval}))
	query, err := runtime.NewQueryOnReader(ctx, zed.NewContext(), program, reader, nil)
	if err != nil {
		return rg, err
	}
	defer query.Close()
	r := query.AsReader()
	for i := 0; i < 2; i++ {
		val, err := r.Read()
		if err != nil {
			return rg, err
		}
		if val == nil {
			break
		}
		if i == 0 {
			rg.Start = val.Deref("lower").AsInt()
			continue
		}
		rg.End = val.Deref("upper").AsInt()
	}
	return rg, nil
}

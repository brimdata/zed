package data

import (
	"context"
	"fmt"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/lake/seekindex"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/runtime/expr/extent"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
)

func LookupSeekRange(ctx context.Context, engine storage.Engine, path *storage.URI,
	obj *Object, cmp expr.CompareFn, filter *expr.SpanFilter, countSpan extent.Span, o order.Which) (*seekindex.Range, error) {
	r, err := engine.Get(ctx, obj.SeekIndexURI(path))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var rg *seekindex.Range
	unmarshaler := zson.NewZNGUnmarshaler()
	reader := zngio.NewReader(zed.NewContext(), r)
	swapper := expr.NewValueCompareFn(order.Asc, o == order.Asc)
	for {
		val, err := reader.Read()
		if val == nil || err != nil {
			return rg, err
		}
		var entry seekindex.Entry
		if err := unmarshaler.Unmarshal(val, &entry); err != nil {
			return nil, fmt.Errorf("corrupt seek index entry for %q at value: %q (%w)", obj.ID.String(), zson.String(val), err)
		}
		from := entry.From
		to := entry.To
		if swapper(from, to) > 0 {
			from, to = to, from
		}
		if filter != nil && filter.Eval(from, to) {
			continue
		}
		if countSpan != nil {
			seqFrom := zed.NewValue(zed.TypeUint64, zed.EncodeUint(entry.ValOff))
			seqTo := zed.NewValue(zed.TypeUint64, zed.EncodeUint(entry.ValOff+entry.ValCnt-1))
			if !countSpan.Overlaps(seqFrom, seqTo) {
				continue
			}
		}
		if rg == nil {
			rg = &seekindex.Range{Offset: int64(entry.Offset)}
		}
		rg.Length += int64(entry.Length)
	}
}

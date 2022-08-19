package seekindex

import (
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/order"
	"github.com/brimdata/zed/runtime/expr"
	"github.com/brimdata/zed/runtime/expr/extent"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zson"
)

type Section struct {
	Keys                  extent.Span
	Range                 Range
	Counts                extent.Span
	FirstCount, LastCount uint64
}

type SectionReader struct {
	cmp       expr.CompareFn
	peeker    *zio.Peeker
	unmarshal *zson.UnmarshalZNGContext
}

func NewSectionReader(r io.Reader, last zed.Value, count uint64, size int64, cmp expr.CompareFn) *SectionReader {
	// Construct last entry and concat it to the stream.
	val, err := zson.MarshalZNG(&Entry{Key: &last, Count: count, Offset: size})
	if err != nil {
		panic(err)
	}
	var reader zio.Reader = zngio.NewReader(zed.NewContext(), r)
	reader = zio.ConcatReader(reader, zbuf.NewArray([]zed.Value{*val}))
	return &SectionReader{
		cmp:       cmp,
		peeker:    zio.NewPeeker(reader),
		unmarshal: zson.NewZNGUnmarshaler(),
	}
}

func (r *SectionReader) Next() (*Section, error) {
	val, err := r.peeker.Read()
	if val == nil || err != nil {
		return nil, err
	}
	var first Entry
	if err := r.unmarshal.Unmarshal(val, &first); err != nil {
		return nil, err
	}
	val, err = r.peeker.Peek()
	if val == nil || err != nil {
		return nil, err
	}
	var last Entry
	if err := r.unmarshal.Unmarshal(val, &last); err != nil {
		return nil, err
	}
	firstCount := *zed.NewValue(zed.TypeUint64, zed.EncodeUint(first.Count))
	lastCount := *zed.NewValue(zed.TypeUint64, zed.EncodeUint(last.Count))
	return &Section{
		Range:  Range{Start: first.Offset, End: last.Offset},
		Keys:   extent.NewGeneric(*first.Key, *last.Key, r.cmp),
		Counts: extent.NewGenericFromOrder(firstCount, lastCount, order.Asc),
	}, nil
}

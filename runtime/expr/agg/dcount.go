package agg

import (
	"fmt"

	"github.com/axiomhq/hyperloglog"
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zson"
)

// DCount uses hyperloglog to approximate the count of unique values for
// a field.
type DCount struct {
	scratch zcode.Bytes
	sketch  *hyperloglog.Sketch
}

var _ Function = (*DCount)(nil)

func NewDCount() *DCount {
	return &DCount{
		sketch: hyperloglog.New(),
	}
}

func (d *DCount) Consume(val *zed.Value) {
	d.scratch = d.scratch[:0]
	// append type id to vals so we get a unique count where the bytes are same
	// but the zed.Type is different.
	d.scratch = zed.AppendInt(d.scratch, int64(val.Type.ID()))
	d.scratch = append(d.scratch, val.Bytes...)
	d.sketch.Insert(d.scratch)
}

func (d *DCount) Result(*zed.Context) *zed.Value {
	return zed.NewValue(zed.TypeUint64, zed.EncodeUint(d.sketch.Estimate()))
}

func (d *DCount) ConsumeAsPartial(partial *zed.Value) {
	if partial.Type != zed.TypeBytes {
		panic(fmt.Errorf("dcount: partial has bad type: %s", zson.MustFormatValue(partial)))
	}
	var s hyperloglog.Sketch
	if err := s.UnmarshalBinary(partial.Bytes); err != nil {
		panic(fmt.Errorf("dcount: unmarshaling partial: %w", err))
	}
	d.sketch.Merge(&s)
}

func (d *DCount) ResultAsPartial(zctx *zed.Context) *zed.Value {
	b, err := d.sketch.MarshalBinary()
	if err != nil {
		panic(fmt.Errorf("dcount: marshaling partial: %w", err))
	}
	return zed.NewBytes(b)
}

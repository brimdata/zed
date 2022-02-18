package agg

import (
	"github.com/axiomhq/hyperloglog"
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
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

func (*DCount) ConsumeAsPartial(*zed.Value) {
	// XXX this is straightforward to do using c.sketch.Merge().  See #1892.
	panic("dcount: partials not yet implemented")
}

func (*DCount) ResultAsPartial(zctx *zed.Context) *zed.Value {
	// XXX this is straightforward to do using c.sketch.Merge().  See #1892.
	panic("dcount: partials not yet implemented")
}

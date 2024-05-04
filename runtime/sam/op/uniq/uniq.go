package uniq

import (
	"bytes"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zcode"
)

type Op struct {
	rctx    *runtime.Context
	parent  zbuf.Puller
	builder zcode.Builder
	cflag   bool
	count   uint64
	last    zed.Value
	batch   zbuf.Batch
	arena   *zed.Arena
}

func New(rctx *runtime.Context, parent zbuf.Puller, cflag bool) *Op {
	return &Op{
		rctx:   rctx,
		parent: parent,
		cflag:  cflag,
	}
}

func (o *Op) wrap(t zed.Value) zed.Value {
	if o.cflag {
		if o.arena == nil {
			o.arena = zed.NewArena()
		}
		o.builder.Reset()
		o.builder.Append(t.Bytes())
		o.builder.Append(zed.EncodeUint(o.count))
		typ := o.rctx.Zctx.MustLookupTypeRecord([]zed.Field{
			zed.NewField("value", t.Type()),
			zed.NewField("count", zed.TypeUint64),
		})
		return o.arena.New(typ, o.builder.Bytes())
	}
	return t
}

func (o *Op) appendUniq(out []zed.Value, t zed.Value) []zed.Value {
	if o.count == 0 {
		o.last = t
		o.count = 1
		return out
	} else if bytes.Equal(t.Bytes(), o.last.Bytes()) {
		o.count++
		return out
	}
	out = append(out, o.wrap(o.last))
	o.last = t
	o.count = 1
	return out
}

// uniq is a little bit complicated because we have to check uniqueness
// across records between calls to Pull.
func (o *Op) Pull(done bool) (zbuf.Batch, error) {
	for {
		batch, err := o.parent.Pull(done)
		if err != nil {
			return nil, err
		}
		if batch == nil {
			if o.count == 0 {
				return nil, nil
			}
			t := o.wrap(o.last)
			o.count = 0
			return zbuf.NewArray(o.arena, []zed.Value{t}), nil
		}
		var out []zed.Value
		vals := batch.Values()
		for i := range vals {
			out = o.appendUniq(out, vals[i])
		}
		if o.batch != nil {
			o.batch.Unref()
		}
		o.batch = batch
		if len(out) > 0 {
			return zbuf.NewBatch(o.arena, out, batch, batch.Vars()), nil
		}
	}
}

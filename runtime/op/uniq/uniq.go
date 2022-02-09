package uniq

import (
	"bytes"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/runtime/op"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zcode"
)

type Proc struct {
	pctx    *op.Context
	parent  zbuf.Puller
	builder zcode.Builder
	cflag   bool
	count   uint64
	last    *zed.Value
}

func New(pctx *op.Context, parent zbuf.Puller, cflag bool) *Proc {
	return &Proc{
		pctx:   pctx,
		parent: parent,
		cflag:  cflag,
	}
}

func (p *Proc) wrap(t *zed.Value) *zed.Value {
	if p.cflag {
		p.builder.Reset()
		p.builder.Append(t.Bytes)
		p.builder.Append(zed.EncodeUint(p.count))
		typ := p.pctx.Zctx.MustLookupTypeRecord([]zed.Column{
			zed.NewColumn("value", t.Type),
			zed.NewColumn("count", zed.TypeUint64),
		})
		return zed.NewValue(typ, p.builder.Bytes()).Copy()
	}
	return t
}

func (p *Proc) appendUniq(out []zed.Value, t *zed.Value) []zed.Value {
	if p.count == 0 {
		p.last = t.Copy()
		p.count = 1
		return out
	} else if bytes.Equal(t.Bytes, p.last.Bytes) {
		p.count++
		return out
	}
	out = append(out, *p.wrap(p.last))
	p.last = t.Copy()
	p.count = 1
	return out
}

// uniq is a little bit complicated because we have to check uniqueness
// across records between calls to Pull.
func (p *Proc) Pull(done bool) (zbuf.Batch, error) {
	for {
		batch, err := p.parent.Pull(done)
		if err != nil {
			return nil, err
		}
		if batch == nil {
			if p.last == nil {
				return nil, nil
			}
			t := p.wrap(p.last)
			p.last = nil
			return zbuf.NewArray([]zed.Value{*t}), nil
		}
		var out []zed.Value
		vals := batch.Values()
		for i := range vals {
			out = p.appendUniq(out, &vals[i])
		}
		batch.Unref()
		if len(out) > 0 {
			return zbuf.NewArray(out), nil
		}
	}
}

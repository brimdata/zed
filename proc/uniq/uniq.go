package uniq

import (
	"bytes"

	"github.com/brimsec/zq/proc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zng"
	"go.uber.org/zap"
)

type Proc struct {
	parent proc.Interface
	ctx    *proc.Context
	cflag  bool
	count  uint64
	last   *zng.Record
}

func New(ctx *proc.Context, parent proc.Interface, cflag bool) *Proc {
	return &Proc{
		parent: parent,
		ctx:    ctx,
		cflag:  cflag,
	}
}

func (p *Proc) wrap(t *zng.Record) *zng.Record {
	if p.cflag {
		cols := []zng.Column{zng.NewColumn("_uniq", zng.TypeUint64)}
		vals := []zng.Value{zng.NewUint64(p.count)}
		newR, err := p.ctx.TypeContext.AddColumns(t, cols, vals)
		if err != nil {
			p.ctx.Logger.Error("AddColumns failed", zap.Error(err))
			return t
		}
		return newR
	}
	return t
}

func (p *Proc) appendUniq(out []*zng.Record, t *zng.Record) []*zng.Record {
	if p.count == 0 {
		p.last = t.Keep()
		p.count = 1
		return out
	} else if bytes.Equal(t.Raw, p.last.Raw) {
		p.count++
		return out
	}
	out = append(out, p.wrap(p.last))
	p.last = t.Keep()
	p.count = 1
	return out
}

// uniq is a little bit complicated because we have to check uniqueness
// across records between calls to Pull.
func (p *Proc) Pull() (zbuf.Batch, error) {
	batch, err := p.parent.Pull()
	if err != nil {
		return nil, err
	}
	if batch == nil {
		if p.last == nil {
			return nil, nil
		}
		t := p.wrap(p.last)
		p.last = nil
		return zbuf.NewArray([]*zng.Record{t}), nil
	}
	defer batch.Unref()
	var out []*zng.Record
	for k := 0; k < batch.Length(); k++ {
		out = p.appendUniq(out, batch.Index(k))
	}
	return zbuf.NewArray(out), nil
}

func (p *Proc) Done() {
	p.parent.Done()
}

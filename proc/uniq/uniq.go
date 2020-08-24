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

func (u *Proc) wrap(t *zng.Record) *zng.Record {
	if u.cflag {
		cols := []zng.Column{zng.NewColumn("_uniq", zng.TypeUint64)}
		vals := []zng.Value{zng.NewUint64(u.count)}
		newR, err := u.ctx.TypeContext.AddColumns(t, cols, vals)
		if err != nil {
			u.ctx.Logger.Error("AddColumns failed", zap.Error(err))
			return t
		}
		return newR
	}
	return t
}

func (u *Proc) appendUniq(out []*zng.Record, t *zng.Record) []*zng.Record {
	if u.count == 0 {
		u.last = t.Keep()
		u.count = 1
		return out
	} else if bytes.Equal(t.Raw, u.last.Raw) {
		u.count++
		return out
	}
	out = append(out, u.wrap(u.last))
	u.last = t.Keep()
	u.count = 1
	return out
}

// uniq is a little bit complicated because we have to check uniqueness
// across records between calls to Pull.
func (u *Proc) Pull() (zbuf.Batch, error) {
	batch, err := u.parent.Pull()
	if err != nil {
		return nil, err
	}
	if batch == nil {
		if u.last == nil {
			return nil, nil
		}
		t := u.wrap(u.last)
		u.last = nil
		return zbuf.NewArray([]*zng.Record{t}), nil
	}
	defer batch.Unref()
	var out []*zng.Record
	for k := 0; k < batch.Length(); k++ {
		out = u.appendUniq(out, batch.Index(k))
	}
	return zbuf.NewArray(out), nil
}

func (p *Proc) Done() {
	p.parent.Done()
}

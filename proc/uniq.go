package proc

import (
	"bytes"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/zbuf"
	"github.com/mccanne/zq/zng"
	"go.uber.org/zap"
)

type Uniq struct {
	Base
	cflag bool
	count uint64
	last  *zng.Record
}

func NewUniq(c *Context, parent Proc, cflag bool) *Uniq {
	return &Uniq{Base: Base{Context: c, Parent: parent}, cflag: cflag}
}

func (u *Uniq) wrap(t *zng.Record) *zng.Record {
	if u.cflag {
		cols := []zng.Column{zng.NewColumn("_uniq", zng.TypeCount)}
		vals := []zng.Value{zng.NewCount(u.count)}
		newR, err := u.TypeContext.AddColumns(t, cols, vals)
		if err != nil {
			u.Logger.Error("AddColumns failed", zap.Error(err))
			return t
		}
		return newR
	}
	return t
}

func (u *Uniq) appendUniq(out []*zng.Record, t *zng.Record) []*zng.Record {
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
func (u *Uniq) Pull() (zbuf.Batch, error) {
	batch, err := u.Get()
	if err != nil {
		return nil, err
	}
	span := nano.NewSpanTs(u.MinTs, u.MaxTs)
	if batch == nil {
		if u.last == nil {
			return nil, nil
		}
		t := u.wrap(u.last)
		u.last = nil
		return zbuf.NewArray([]*zng.Record{t}, span), nil
	}
	defer batch.Unref()
	var out []*zng.Record
	for k := 0; k < batch.Length(); k++ {
		out = u.appendUniq(out, batch.Index(k))
	}
	return zbuf.NewArray(out, span), nil
}

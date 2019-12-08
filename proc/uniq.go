package proc

import (
	"bytes"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zq"
	"go.uber.org/zap"
)

type Uniq struct {
	Base
	cflag bool
	count uint64
	last  *zq.Record
}

func NewUniq(c *Context, parent Proc, cflag bool) *Uniq {
	return &Uniq{Base: Base{Context: c, Parent: parent}, cflag: cflag}
}

func (u *Uniq) wrap(t *zq.Record) *zq.Record {
	if u.cflag {
		cols := []zeek.Column{{Name: "_uniq", Type: zeek.TypeCount}}
		vals := []zeek.Value{zeek.NewCount(u.count)}
		newR, err := u.Resolver.AddColumns(t, cols, vals)
		if err != nil {
			u.Logger.Error("AddColumns failed", zap.Error(err))
			return t
		}
		return newR
	}
	return t
}

func (u *Uniq) appendUniq(out []*zq.Record, t *zq.Record) []*zq.Record {
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
func (u *Uniq) Pull() (zq.Batch, error) {
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
		return zq.NewArray([]*zq.Record{t}, span), nil
	}
	defer batch.Unref()
	var out []*zq.Record
	for k := 0; k < batch.Length(); k++ {
		out = u.appendUniq(out, batch.Index(k))
	}
	return zq.NewArray(out, span), nil
}

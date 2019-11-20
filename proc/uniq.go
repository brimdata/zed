package proc

import (
	"bytes"
	"strconv"

	"github.com/mccanne/zq/pkg/nano"
	"github.com/mccanne/zq/pkg/zeek"
	"github.com/mccanne/zq/pkg/zson"
	"go.uber.org/zap"
)

type Uniq struct {
	Base
	cflag bool
	count int
	last  *zson.Record
}

func NewUniq(c *Context, parent Proc, cflag bool) *Uniq {
	return &Uniq{Base: Base{Context: c, Parent: parent}, cflag: cflag}
}

func (u *Uniq) wrap(t *zson.Record) *zson.Record {
	if u.cflag {
		cols := []zeek.Column{{Name: "_uniq", Type: zeek.TypeCount}}
		vals := []string{strconv.FormatInt(int64(u.count), 10)}
		newR, err := u.Resolver.AddColumns(t, cols, vals)
		if err != nil {
			u.Logger.Error("AddColumns failed", zap.Error(err))
			return t
		}
		return newR
	}
	return t
}

func (u *Uniq) appendUniq(out []*zson.Record, t *zson.Record) []*zson.Record {
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
func (u *Uniq) Pull() (zson.Batch, error) {
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
		return zson.NewArray([]*zson.Record{t}, span), nil
	}
	defer batch.Unref()
	var out []*zson.Record
	for k := 0; k < batch.Length(); k++ {
		out = u.appendUniq(out, batch.Index(k))
	}
	return zson.NewArray(out, span), nil
}

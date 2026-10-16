package op

import (
	"github.com/superdb/super"
	"github.com/superdb/super/sbuf"
	"github.com/superdb/super/scode"
	"github.com/superdb/super/vector"
	"github.com/superdb/super/vector/vio"
)

type Uniq struct {
	sctx      *super.Context
	parent    vio.Puller
	withCount bool

	count       int64
	eos         bool
	last        super.Value
	pullBuilder scode.Builder
	wrapBuilder scode.Builder
}

func NewUniq(sctx *super.Context, parent vio.Puller, withCount bool) *Uniq {
	return &Uniq{
		sctx:      sctx,
		parent:    parent,
		withCount: withCount,
	}
}

func (u *Uniq) Pull(done bool) (vector.Any, error) {
	if u.eos {
		u.eos = false
		return nil, nil
	}
	b := vector.NewDynamicValueBuilder()
	for {
		vec, err := u.parent.Pull(done)
		if err != nil {
			return nil, err
		}
		if vec == nil {
			if u.count == 0 {
				return nil, nil
			}
			val := u.wrap(u.last)
			u.count = 0
			u.eos = true
			// The zero value is not a valid super.Value.
			u.last = super.Value{}
			return sbuf.Dematerialize(u.sctx, val), nil
		}
		for i := range vec.Len() {
			u.writeUniq(b, vector.ValueAt(&u.pullBuilder, vec, i))
		}
		if vec := b.Build(u.sctx); vec.Len() > 0 {
			return vec, nil
		}
	}
}

func (u *Uniq) writeUniq(b *vector.DynamicValueBuilder, val super.Value) {
	if u.count == 0 {
		u.count = 1
		u.last = val.Copy()
	} else if val.Equal(u.last) {
		u.count++
	} else {
		b.Write(u.wrap(u.last))
		u.count = 1
		u.last = val.Copy()
	}
}

func (u *Uniq) wrap(val super.Value) super.Value {
	if !u.withCount {
		return val
	}
	b := u.wrapBuilder
	b.Truncate()
	b.Append(val.Bytes())
	b.Append(super.EncodeInt(u.count))
	typ := u.sctx.MustLookupTypeRecord([]super.Field{
		super.NewField("value", val.Type()),
		super.NewField("count", super.TypeInt64),
	})
	return super.NewValue(typ, b.Bytes())
}

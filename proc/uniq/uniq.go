package uniq

import (
	"bytes"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/proc"
	"github.com/brimdata/zed/zbuf"
	"go.uber.org/zap"
)

type Proc struct {
	pctx   *proc.Context
	parent proc.Interface
	cflag  bool
	count  uint64
	last   *zed.Value
}

func New(pctx *proc.Context, parent proc.Interface, cflag bool) *Proc {
	return &Proc{
		pctx:   pctx,
		parent: parent,
		cflag:  cflag,
	}
}

func (p *Proc) wrap(t *zed.Value) *zed.Value {
	if p.cflag {
		// The leading underscore in "_uniq" is to avoid clashing with existing field
		// names. Reducers don't have this problem since Zed has a way to assign
		// a field name to their returned result. At some point we could maybe add an
		// option like "-f foo" to set a field name, at which point we could safely
		// use a non-underscore field name by default, such as "count".
		cols := []zed.Column{zed.NewColumn("_uniq", zed.TypeUint64)}
		vals := []zed.Value{zed.NewUint64(p.count)}
		newR, err := p.pctx.Zctx.AddColumns(t, cols, vals)
		if err != nil {
			p.pctx.Logger.Error("AddColumns failed", zap.Error(err))
			return t
		}
		return newR
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
func (p *Proc) Pull() (zbuf.Batch, error) {
	for {
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
			return zbuf.Array{*t}, nil
		}
		var out []zed.Value
		zvals := batch.Values()
		for i := range zvals {
			out = p.appendUniq(out, &zvals[i])
		}
		batch.Unref()
		if len(out) > 0 {
			return zbuf.Array(out), nil
		}
	}
}

func (p *Proc) Done() {
	p.parent.Done()
}

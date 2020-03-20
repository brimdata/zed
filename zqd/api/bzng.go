package api

import (
	"bytes"
	"context"
	"io"

	"github.com/brimsec/zq/pkg/nano"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio/bzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type BzngSearch struct {
	reader *bzngio.Reader
	cancel context.CancelFunc
	ctrl   interface{}
	err    error
}

func NewBzngSearch(body io.Reader, cancel context.CancelFunc) *BzngSearch {
	if cancel == nil {
		cancel = func() {}
	}
	return &BzngSearch{
		reader: bzngio.NewReader(body, resolver.NewContext()),
		cancel: cancel,
	}
}

//XXX
const mtu = 100

func (r *BzngSearch) Pull() (zbuf.Batch, interface{}, error) {
	v := r.ctrl
	if v != nil {
		r.ctrl = nil
		return nil, v, nil
	}
	minTs, maxTs := nano.MaxTs, nano.MinTs
	var out []*zng.Record
	for len(out) < mtu {
		rec, b, err := r.reader.ReadPayload()
		if err != nil {
			r.cancel()
			return nil, nil, err
		}
		if rec == nil && b == nil {
			break
		}
		if b != nil {
			if !bytes.HasPrefix(b, []byte("json:")) {
				// We expect only json control payloads.
				// XXX should log error if something else,
				// but just skip for now.
				continue
			}
			ctrl, err := unpack(b[5:])
			if err != nil {
				r.cancel()
				return nil, nil, err
			}
			if end, ok := v.(*TaskEnd); ok {
				// We remember the error to return at end of connection.
				// We return nil error for the TaskEnd message
				// even if it contains an error, then we return
				// that error at end of connection.
				r.err = end.Error
			}
			if len(out) > 0 {
				// Save this control message for the next Read
				// and return the current batch.
				r.ctrl = ctrl
				break
			}
			return nil, ctrl, nil
		}
		if rec.Ts < minTs {
			minTs = rec.Ts
		}
		if rec.Ts > maxTs {
			maxTs = rec.Ts
		}
		rec = rec.Keep()
		out = append(out, rec)
	}
	if len(out) > 0 {
		return zbuf.NewArray(out, nano.NewSpanTs(minTs, maxTs)), nil, nil
	}
	r.cancel()
	return nil, nil, r.err
}

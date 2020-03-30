package api

import (
	"bytes"
	"io"

	"github.com/brimsec/zq/zio/bzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

type BzngSearch struct {
	reader *bzngio.Reader
	onctrl func(interface{})
}

func NewBzngSearch(body io.Reader) *BzngSearch {
	return &BzngSearch{
		reader: bzngio.NewReader(body, resolver.NewContext()),
	}
}

// SetOnCtrl registers a callback function that will be fired when a control
// payload is found in the search stream. Not safe for concurrent use, this
// should be set before the first read is called.
func (r *BzngSearch) SetOnCtrl(cb func(interface{})) {
	r.onctrl = cb
}

func (r *BzngSearch) Read() (*zng.Record, error) {
	for {
		rec, b, err := r.reader.ReadPayload()
		if err != nil || b == nil {
			return rec, err
		}
		if !bytes.HasPrefix(b, []byte("json:")) {
			// We expect only json control payloads.
			// XXX should log error if something else,
			// but just skip for now.
			continue
		}
		ctrl, err := unpack(b[5:])
		if err != nil {
			return nil, err
		}
		if r.onctrl != nil {
			r.onctrl(ctrl)
		}
		if end, ok := ctrl.(*TaskEnd); ok && end.Error != nil {
			return nil, end.Error
		}
	}
}

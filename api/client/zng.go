package client

import (
	"io"

	"github.com/brimdata/zed/api"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

type ZngSearch struct {
	reader *zngio.Reader
	onctrl func(interface{})
}

func NewZngSearch(body io.Reader) *ZngSearch {
	return &ZngSearch{
		reader: zngio.NewReader(body, zson.NewContext()),
	}
}

// SetOnCtrl registers a callback function that will be fired when a control
// payload is found in the search stream. Not safe for concurrent use, this
// should be set before the first read is called.
func (r *ZngSearch) SetOnCtrl(cb func(interface{})) {
	r.onctrl = cb
}

func (r *ZngSearch) Read() (*zng.Record, error) {
	for {
		rec, msg, err := r.reader.ReadPayload()
		if err != nil || msg == nil {
			return rec, err
		}
		if msg.Encoding != zng.AppEncodingJSON {
			// We expect only json control payloads.
			// XXX should log error if something else,
			// but just skip for now.
			continue
		}
		ctrl, err := unpack(msg.Bytes)
		if err != nil {
			return nil, err
		}
		if r.onctrl != nil {
			r.onctrl(ctrl)
		}
		if end, ok := ctrl.(*api.TaskEnd); ok && end.Error != nil {
			return nil, end.Error
		}
	}
}

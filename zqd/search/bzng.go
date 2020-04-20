package search

import (
	"encoding/json"
	"net/http"

	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/bzngio"
)

// BzngOutput writes bzng encodings directly to the client via
// binary data sent over http chunked encoding interleaved with json
// protocol messages sent as zng comment payloads.  The simplicity of
// this is a thing of beauty.
// Also, it implements the Output interface.
type BzngOutput struct {
	response http.ResponseWriter
	writer   *bzngio.Writer
}

func NewBzngOutput(response http.ResponseWriter) *BzngOutput {
	o := &BzngOutput{
		response: response,
		writer:   bzngio.NewWriter(response, zio.WriterFlags{}),
	}
	return o
}

func (r *BzngOutput) flush() {
	r.response.(http.Flusher).Flush()
}

func (r *BzngOutput) Collect() interface{} {
	return "TBD" //XXX
}

func (r *BzngOutput) SendBatch(cid int, batch zbuf.Batch) error {
	for _, rec := range batch.Records() {
		// XXX need to send channel id as control payload
		if err := r.writer.Write(rec); err != nil {
			return err
		}
	}
	batch.Unref()
	r.flush()
	return nil
}

func (r *BzngOutput) End(ctrl interface{}) error {
	return r.SendControl(ctrl)
}

func (r *BzngOutput) SendControl(ctrl interface{}) error {
	msg, err := json.Marshal(ctrl)
	if err != nil {
		//XXX need a better json error message
		return err
	}
	b := []byte("json:")
	if err := r.writer.WriteControl(append(b, msg...)); err != nil {
		return err
	}
	r.flush()
	return nil
}

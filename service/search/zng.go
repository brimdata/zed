package search

import (
	"encoding/json"
	"net/http"

	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zng"
)

// ZngOutput writes zng encodings directly to the client via
// binary data sent over http chunked encoding interleaved with json
// protocol messages sent as zng comment payloads.  The simplicity of
// this is a thing of beauty.
// Also, it implements the Output interface.
type ZngOutput struct {
	response http.ResponseWriter
	writer   *zngio.Writer
	ctrl     bool
}

func NewZngOutput(response http.ResponseWriter, ctrl bool) *ZngOutput {
	return &ZngOutput{
		response: response,
		writer: zngio.NewWriter(zio.NopCloser(response), zngio.WriterOpts{
			LZ4BlockSize: zngio.DefaultLZ4BlockSize,
		}),
		ctrl: ctrl,
	}
}

func (r *ZngOutput) flush() {
	r.response.(http.Flusher).Flush()
}

func (r *ZngOutput) Collect() interface{} {
	return "TBD" //XXX
}

func (r *ZngOutput) SendBatch(cid int, batch zbuf.Batch) error {
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

func (r *ZngOutput) End(ctrl interface{}) error {
	if err := r.SendControl(ctrl); err != nil {
		return err
	}
	return r.writer.Close()
}

func (r *ZngOutput) SendControl(ctrl interface{}) error {
	if !r.ctrl {
		return nil
	}
	msg, err := json.Marshal(ctrl)
	if err != nil {
		//XXX need a better json error message
		return err
	}
	if err := r.writer.WriteControl(msg, zng.AppEncodingJSON); err != nil {
		return err
	}
	r.flush()
	return nil
}

func (r *ZngOutput) ContentType() string {
	return MimeTypeZNG
}

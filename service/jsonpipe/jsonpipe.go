package jsonpipe

import (
	"encoding/json"
	"net/http"

	"github.com/brimdata/zed/api"
)

var sep = []byte("\n\n")

// JSONPipe is an abstraction for sending reponses as multiple payloads over
// a potentially long-lived connection using HTTP chunked encoding and a convention
// to separate JSON objects using two newline characters as described in
// https://github.com/eBay/jsonpipe
type JSONPipe struct {
	http.ResponseWriter
	encoder   *json.Encoder
	separator []byte
}

// New creates a new JSONPipe object for streaming the response to
// the indicated request.  The Start should be called to initiate the pipe.
// Then JSON objects are transmitted by calling the Send method one or more times.
// The pipe is closed by calling the End method.
func New(w http.ResponseWriter) *JSONPipe {
	p := &JSONPipe{
		ResponseWriter: w,
		encoder:        json.NewEncoder(w),
		separator:      sep,
	}
	return p
}

func (p *JSONPipe) flush() {
	flusher := p.ResponseWriter.(http.Flusher)
	flusher.Flush()
}

func (p *JSONPipe) SendStart(taskID int64) error {
	return p.Send(api.TaskStart{Type: "TaskStart", TaskID: taskID})
}

func (p *JSONPipe) SendEnd(taskID int64, err error) error {
	var apierr *api.Error
	if err != nil {
		apierr = &api.Error{Type: "Error", Message: err.Error()}
	}
	return p.SendFinal(api.TaskEnd{Type: "TaskEnd", TaskID: taskID, Error: apierr})
}

// Send encodes as JSON the payload and streams it as a message over the
// underlying HTTP connection.  Returns an errorr and does not transmit the message
// if the connection has already been set into an error state.
func (p *JSONPipe) Send(payload interface{}) error {
	if err := p.encoder.Encode(payload); err != nil {
		return err
	}
	if _, err := p.ResponseWriter.Write(p.separator); err != nil {
		return err
	}
	p.flush()
	return nil
}

func (p *JSONPipe) SendFinal(payload interface{}) error {
	defer p.flush()
	return p.encoder.Encode(payload)
}

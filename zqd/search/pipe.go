package search

import (
	"encoding/json"
	"net/http"
)

// JSONPipe is an abstraction for sending reponses as multiple payloads over
// a potentially long-lived connection using HTTP chunked encoding and a convention
// to separate JSON objects using two newline characters as described in
// https://github.com/eBay/jsonpipe
type JSONPipe struct {
	*http.Request
	http.ResponseWriter
	encoder   *json.Encoder
	separator []byte
}

// NewJSONPipe creates a new JSONPipe object for streaming the response to
// the indicated request.  The Start should be called to initiate the pipe.
// Then JSON objects are transmitted by calling the Send method one or more times.
// The pipe is closed by calling the End method.
func NewJSONPipe(r *http.Request, w http.ResponseWriter) *JSONPipe {
	r.Header.Add("Content-Type", "application/x-ndjson")
	p := &JSONPipe{
		Request:        r,
		ResponseWriter: w,
		encoder:        json.NewEncoder(w),
		separator:      []byte("\n\n"),
	}
	return p
}

func (p *JSONPipe) flush() {
	flusher := p.ResponseWriter.(http.Flusher)
	flusher.Flush()
}

// Send encodes as JSON the payload and streams it as a message over the
// underlying HTTP connection.  Returns an errorr and does not transmit the message
// if the connection has already been set into an error state.
func (p *JSONPipe) Send(payload interface{}) error {
	if err := p.Context().Err(); err != nil {
		return err
	}
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
	return p.encoder.Encode(payload)
}

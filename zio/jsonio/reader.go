package jsonio

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zson"
)

type Reader struct {
	zctx       *zed.Context
	decoder    *json.Decoder
	encoder    *json.Encoder
	encoderBuf *bytes.Buffer
}

func NewReader(r io.Reader, zctx *zed.Context) *Reader {
	d := json.NewDecoder(r)
	// Prime d's buffer so we can check for an array.
	d.More()
	var b [1]byte
	if n, _ := d.Buffered().Read(b[:]); n > 0 && b[0] == '[' {
		// We have an array.  Discard its opening "[" delimiter.
		d.Token()
	}
	var buf bytes.Buffer
	e := json.NewEncoder(&buf)
	e.SetEscapeHTML(false)
	return &Reader{
		zctx:       zctx,
		decoder:    d,
		encoder:    e,
		encoderBuf: &buf,
	}
}

func (r *Reader) Read() (*zed.Record, error) {
	if !r.decoder.More() {
		return nil, nil
	}
	var v interface{}
	if err := r.decoder.Decode(&v); err != nil {
		return nil, err
	}
	if _, ok := v.(map[string]interface{}); !ok {
		v = map[string]interface{}{"value": v}
	}
	r.encoderBuf.Reset()
	if err := r.encoder.Encode(v); err != nil {
		return nil, err
	}
	return zson.NewReader(r.encoderBuf, r.zctx).Read()
}

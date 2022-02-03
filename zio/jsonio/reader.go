package jsonio

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio/zsonio"
)

type Reader struct {
	zctx       *zed.Context
	decoder    *json.Decoder
	encoder    *json.Encoder
	encoderBuf *bytes.Buffer
}

func NewReader(r io.Reader, zctx *zed.Context) *Reader {
	var buf bytes.Buffer
	e := json.NewEncoder(&buf)
	e.SetEscapeHTML(false)
	return &Reader{
		zctx:       zctx,
		decoder:    json.NewDecoder(r),
		encoder:    e,
		encoderBuf: &buf,
	}
}

func (r *Reader) Read() (*zed.Value, error) {
	if !r.decoder.More() {
		return nil, nil
	}
	var v interface{}
	if err := r.decoder.Decode(&v); err != nil {
		return nil, err
	}
	r.encoderBuf.Reset()
	if err := r.encoder.Encode(v); err != nil {
		return nil, err
	}
	return zsonio.NewReader(r.encoderBuf, r.zctx).Read()
}

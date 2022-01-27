package jsonio

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio/zsonio"
)

type Reader struct {
	zctx    *zed.Context
	decoder *json.Decoder
	buf     json.RawMessage
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
	return &Reader{
		zctx:    zctx,
		decoder: d,
	}
}

func (r *Reader) Read() (*zed.Value, error) {
	if !r.decoder.More() {
		return nil, nil
	}
	if err := r.decoder.Decode(&r.buf); err != nil {
		return nil, err
	}
	zr := zsonio.NewReader(bytes.NewReader(r.buf), r.zctx)
	zr.JSONStrict()
	return zr.Read()
}

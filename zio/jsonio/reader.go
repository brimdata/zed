package jsonio

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

type Reader struct {
	zctx    *zson.Context
	decoder *json.Decoder
}

func NewReader(r io.Reader, zctx *zson.Context) *Reader {
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

func (r *Reader) Read() (*zng.Record, error) {
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
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return zson.NewReader(bytes.NewReader(b), r.zctx).Read()
}

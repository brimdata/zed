package jsonio

import (
	"encoding/json"
	"io"

	"github.com/brimdata/zed"
)

type Writer struct {
	io.Closer
	encoder *json.Encoder
}

func NewWriter(wc io.WriteCloser) *Writer {
	return &Writer{
		Closer:  wc,
		encoder: json.NewEncoder(wc),
	}
}

func (w *Writer) Write(val *zed.Value) error {
	return w.encoder.Encode(marshalAny(val.Type, val.Bytes))
}

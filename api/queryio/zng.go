package queryio

import (
	"bytes"
	"fmt"
	"io"

	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

type ZNGWriter struct {
	*zngio.Writer
	marshaler *zson.MarshalZNGContext
}

func NewZNGWriter(w io.WriteCloser) *ZNGWriter {
	m := zson.NewZNGMarshaler()
	m.Decorate(zson.StyleSimple)
	return &ZNGWriter{
		Writer: zngio.NewWriter(w, zngio.WriterOpts{
			LZ4BlockSize: zngio.DefaultLZ4BlockSize,
		}),
		marshaler: m,
	}
}

func (w *ZNGWriter) WriteControl(v interface{}) error {
	rec, err := w.marshaler.MarshalRecord(v)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	err = zsonio.NewWriter(zio.NopCloser(&buf), zsonio.WriterOpts{}).Write(rec)
	if err != nil {
		return err
	}
	return w.Writer.WriteControl(buf.Bytes(), zng.AppEncodingZSON)
}

type ZNGReader struct {
	reader *zngio.Reader
}

func NewZNGReader(r *zngio.Reader) *ZNGReader {
	return &ZNGReader{
		reader: r,
	}
}

func (r *ZNGReader) ReadPayload() (*zng.Record, interface{}, error) {
	rec, msg, err := r.reader.ReadPayload()
	if msg != nil {
		if msg.Encoding != zng.AppEncodingZSON {
			return nil, nil, fmt.Errorf("unsupported app encoding: %v", msg.Encoding)
		}
		value, err := zson.ParseValue(zson.NewContext(), string(msg.Bytes))
		if err != nil {
			return nil, nil, err
		}
		var v interface{}
		if err := unmarshaler.Unmarshal(value, &v); err != nil {
			return nil, nil, err
		}
		return nil, v, nil
	}
	return rec, nil, err
}

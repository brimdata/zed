package queryio

import (
	"bytes"
	"io"

	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/brimdata/zed/zson"
)

type ZNGWriter struct {
	*zngio.Writer
	marshaler *zson.MarshalZNGContext
}

var _ controlWriter = (*ZJSONWriter)(nil)

func NewZNGWriter(w io.Writer) *ZNGWriter {
	m := zson.NewZNGMarshaler()
	m.Decorate(zson.StyleSimple)
	return &ZNGWriter{
		Writer:    zngio.NewWriter(zio.NopCloser(w)),
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
	return w.Writer.WriteControl(buf.Bytes(), zngio.ControlFormatZSON)
}

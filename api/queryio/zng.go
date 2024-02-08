package queryio

import (
	"bytes"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/zngio"
	"github.com/brimdata/zed/zio/zsonio"
	"github.com/brimdata/zed/zson"
)

type ZNGWriter struct {
	*zngio.Writer
	arena     *zed.Arena
	marshaler *zson.MarshalZNGContext
}

var _ controlWriter = (*ZJSONWriter)(nil)

func NewZNGWriter(w io.Writer) *ZNGWriter {
	arena := zed.NewArena(zed.NewContext())
	m := zson.NewZNGMarshalerWithContext(arena.Zctx())
	m.Decorate(zson.StyleSimple)
	return &ZNGWriter{
		Writer:    zngio.NewWriter(zio.NopCloser(w)),
		arena:     arena,
		marshaler: m,
	}
}

func (w *ZNGWriter) WriteControl(v interface{}) error {
	w.arena.Reset()
	val, err := w.marshaler.Marshal(w.arena, v)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	err = zsonio.NewWriter(zio.NopCloser(&buf), zsonio.WriterOpts{}).Write(val)
	if err != nil {
		return err
	}
	return w.Writer.WriteControl(buf.Bytes(), zngio.ControlFormatZSON)
}

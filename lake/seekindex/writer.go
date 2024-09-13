package seekindex

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
)

type Writer struct {
	marshal *zson.MarshalZNGContext
	arena   *zed.Arena
	writer  zio.WriteCloser
	offset  uint64
	valoff  uint64
}

func NewWriter(w zio.WriteCloser) *Writer {
	return &Writer{
		marshal: zson.NewZNGMarshaler(),
		arena:   zed.NewArena(),
		writer:  w,
	}
}

func (w *Writer) Write(min, max zed.Value, valoff uint64, offset uint64) error {
	w.arena.Reset()
	val, err := w.marshal.Marshal(w.arena, &Entry{
		Min:    min,
		Max:    max,
		ValOff: w.valoff,
		ValCnt: valoff - w.valoff,
		Offset: w.offset,
		Length: offset - w.offset,
	})
	w.valoff = valoff
	w.offset = offset
	if err != nil {
		return err
	}
	return w.writer.Write(val)
}

func (w *Writer) Close() error { return w.writer.Close() }

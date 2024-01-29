package seekindex

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
)

type Entry struct {
	Min    zed.Value `zed:"min"`
	Max    zed.Value `zed:"max"`
	ValOff uint64    `zed:"val_off"`
	ValCnt uint64    `zed:"val_cnt"`
	Offset uint64    `zed:"offset"`
	Length uint64    `zed:"length"`
}

type Writer struct {
	arena   *zed.Arena
	marshal *zson.MarshalZNGContext
	writer  zio.WriteCloser
	offset  uint64
	valoff  uint64
}

func NewWriter(w zio.WriteCloser) *Writer {
	zctx := zed.NewContext()
	return &Writer{
		arena:   zed.NewArena(zctx),
		marshal: zson.NewZNGMarshalerWithContext(zctx),
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

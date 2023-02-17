package seekindex

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zson"
)

type Entry struct {
	From   *zed.Value `zed:"from"`
	To     *zed.Value `zed:"to"`
	ValOff uint64     `zed:"val_off"`
	ValCnt uint64     `zed:"val_cnt"`
	Offset uint64     `zed:"offset"`
	Length uint64     `zed:"length"`
}

type Writer struct {
	marshal *zson.MarshalZNGContext
	writer  zio.Writer
	offset  uint64
	valoff  uint64
}

func NewWriter(w zio.Writer) *Writer {
	return &Writer{
		marshal: zson.NewZNGMarshaler(),
		writer:  w,
	}
}

func (w *Writer) Write(from, to *zed.Value, valoff uint64, offset uint64) error {
	val, err := w.marshal.Marshal(&Entry{
		From:   from,
		To:     to,
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
	return w.writer.Write(val.Copy())
}

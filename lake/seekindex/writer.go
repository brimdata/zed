package seekindex

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio"
)

type Entry struct {
	Key    *zed.Value `zed:"key"`
	Count  uint64     `zed:"count"`
	Offset int64      `zed:"offset"`
}

type Writer struct {
	zctx    *zed.Context
	builder *zcode.Builder
	writer  zio.Writer
	typ     zed.Type
	recType *zed.TypeRecord
}

func NewWriter(w zio.Writer) *Writer {
	return &Writer{
		zctx:    zed.NewContext(),
		builder: zcode.NewBuilder(),
		writer:  w,
	}
}

func (w *Writer) Write(key zed.Value, count uint64, offset int64) error {
	b := w.builder
	b.Truncate()
	b.Append(key.Bytes)
	b.Append(zed.EncodeUint(count))
	b.Append(zed.EncodeInt(offset))
	if w.typ != key.Type {
		var schema = []zed.Column{
			{Name: "key", Type: key.Type},
			{Name: "count", Type: zed.TypeUint64},
			{Name: "offset", Type: zed.TypeInt64},
		}
		w.recType = w.zctx.MustLookupTypeRecord(schema)
		w.typ = key.Type
	}
	return w.writer.Write(zed.NewValue(w.recType, b.Bytes()))
}

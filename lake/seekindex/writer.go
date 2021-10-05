package seekindex

import (
	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio"
)

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

func (w *Writer) Write(key zed.Value, offset int64) error {
	b := w.builder
	b.Reset()
	if zed.IsContainerType(key.Type) {
		b.AppendContainer(key.Bytes)
	} else {
		b.AppendPrimitive(key.Bytes)
	}
	b.AppendPrimitive(zed.EncodeInt(offset))
	if w.typ != key.Type {
		var schema = []zed.Column{
			{"key", key.Type},
			{"offset", zed.TypeInt64},
		}
		w.recType = w.zctx.MustLookupTypeRecord(schema)
		w.typ = key.Type
	}
	return w.writer.Write(zed.NewRecord(w.recType, b.Bytes()))
}

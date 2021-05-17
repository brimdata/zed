package seekindex

import (
	"github.com/brimdata/zed/zcode"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

type Writer struct {
	zctx    *zson.Context
	builder *zcode.Builder
	writer  zio.Writer
	typ     zng.Type
	recType *zng.TypeRecord
}

func NewWriter(w zio.Writer) *Writer {
	return &Writer{
		zctx:    zson.NewContext(),
		builder: zcode.NewBuilder(),
		writer:  w,
	}
}

func (w *Writer) Write(key zng.Value, offset int64) error {
	b := w.builder
	b.Reset()
	if zng.IsContainerType(key.Type) {
		b.AppendContainer(key.Bytes)
	} else {
		b.AppendPrimitive(key.Bytes)
	}
	b.AppendPrimitive(zng.EncodeInt(offset))
	if w.typ != key.Type {
		var schema = []zng.Column{
			{"key", key.Type},
			{"offset", zng.TypeInt64},
		}
		w.recType = w.zctx.MustLookupTypeRecord(schema)
		w.typ = key.Type
	}
	return w.writer.Write(zng.NewRecord(w.recType, b.Bytes()))
}

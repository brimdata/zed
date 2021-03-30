package parquetio

import (
	"io"

	"github.com/brimdata/zed/zio/csvio"
	"github.com/brimdata/zed/zng"
	goparquet "github.com/fraugster/parquet-go"
)

type Writer struct {
	w io.WriteCloser

	fw  *goparquet.FileWriter
	typ *zng.TypeRecord
}

func NewWriter(w io.WriteCloser) *Writer {
	return &Writer{w: w}
}

func (w *Writer) Close() error {
	var err error
	if w.fw != nil {
		err = w.fw.Close()
	}
	if err2 := w.w.Close(); err == nil {
		err = err2
	}
	return err
}

func (w *Writer) Write(rec *zng.Record) error {
	if w.typ == nil {
		w.typ = zng.TypeRecordOf(rec.Type)
		sd, err := newSchemaDefinition(w.typ)
		if err != nil {
			return err
		}
		w.fw = goparquet.NewFileWriter(w.w, goparquet.WithSchemaDefinition(sd))
	} else if w.typ != rec.Type {
		return csvio.ErrNotDataFrame
	}
	data, err := newRecordData(zng.TypeRecordOf(rec.Type), rec.Bytes)
	if err != nil {
		return err
	}
	return w.fw.AddData(data)
}

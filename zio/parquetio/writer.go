package parquetio

import (
	"errors"
	"fmt"
	"io"

	"github.com/brimdata/zed"
	"github.com/brimdata/zed/zson"
	goparquet "github.com/fraugster/parquet-go"
	"github.com/fraugster/parquet-go/parquet"
)

type Writer struct {
	w io.WriteCloser

	fw  *goparquet.FileWriter
	typ *zed.TypeRecord
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

func (w *Writer) Write(rec *zed.Value) error {
	recType, ok := zed.TypeUnder(rec.Type).(*zed.TypeRecord)
	if !ok {
		return fmt.Errorf("Parquet output encountered non-record value: %s", zson.String(rec))
	}
	if w.typ == nil {
		w.typ = recType
		sd, err := newSchemaDefinition(recType)
		if err != nil {
			return err
		}
		w.fw = goparquet.NewFileWriter(w.w,
			goparquet.WithCompressionCodec(parquet.CompressionCodec_SNAPPY),
			goparquet.WithSchemaDefinition(sd))
	} else if w.typ != recType {
		return errors.New(
			"Parquet output requires uniform records but multiple types encountered (consider 'fuse')")
	}
	data, err := newRecordData(recType, rec.Bytes)
	if err != nil {
		return err
	}
	return w.fw.AddData(data)
}

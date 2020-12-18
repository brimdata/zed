package zsonio

import (
	"io"

	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zson"
)

type Writer struct {
	writer    io.WriteCloser
	formatter *zson.Formatter
}

type WriterOpts struct {
	Pretty int
}

func NewWriter(w io.WriteCloser, opts WriterOpts) *Writer {
	return &Writer{
		formatter: zson.NewFormatter(opts.Pretty),
		writer:    w,
	}
}

func (w *Writer) Close() error {
	return w.writer.Close()
}

func (w *Writer) Write(rec *zng.Record) error {
	s, err := w.formatter.FormatRecord(rec)
	if err != nil {
		return err
	}
	if _, err := w.writer.Write([]byte(s)); err != nil {
		return err
	}
	_, err = w.writer.Write([]byte("\n"))
	return err
}

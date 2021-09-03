package zsonio

import (
	"io"
	"regexp"

	"github.com/brimdata/zed/zng"
	"github.com/brimdata/zed/zson"
)

type Writer struct {
	writer    io.WriteCloser
	formatter *zson.Formatter
}

type WriterOpts struct {
	Pretty  int
	Persist *regexp.Regexp
}

func NewWriter(w io.WriteCloser, opts WriterOpts) *Writer {
	return &Writer{
		formatter: zson.NewFormatter(opts.Pretty, opts.Persist),
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
	if _, err := io.WriteString(w.writer, s); err != nil {
		return err
	}
	_, err = w.writer.Write([]byte("\n"))
	return err
}

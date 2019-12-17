package zsio

import (
	"io"

	"github.com/mccanne/zq/pkg/zson"
)

type Flags struct {
	UTF8       bool
	ShowTypes  bool
	ShowFields bool
	EpochDates bool
}

type Writer struct {
	zson.WriteFlusher
	io.Closer
}

func NewWriter(writer zson.WriteFlusher, closer io.Closer) *Writer {
	return &Writer{
		WriteFlusher: writer,
		Closer:       closer,
	}
}

func (w *Writer) Close() error {
	err := w.Flush()
	cerr := w.Closer.Close()
	if err == nil {
		err = cerr
	}
	return err
}

func Extension(format string) string {
	switch format {
	case "zson":
		return ".zson"
	case "zeek":
		return ".log"
	case "ndjson":
		return ".ndjson"
	case "zjson":
		return ".ndjson"
	case "text":
		return ".txt"
	case "table":
		return ".tbl"
	case "bzson":
		return ".bzson"
	default:
		return ""
	}
}

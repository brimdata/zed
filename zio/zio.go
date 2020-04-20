package zio

import (
	"flag"
	"io"

	"github.com/brimsec/zq/zbuf"
)

// ReaderFlags has the union of the flags accepted by all the different
// Reader implementations.
type ReaderFlags struct {
	Format string
}

func (f *ReaderFlags) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&f.Format, "i", "auto", "format of input data [auto,bzng,ndjson,zeek,zjson,zng]")
}

// WriterFlags has the union of the flags accepted by all the different
// Writer implementations.
type WriterFlags struct {
	Format           string
	UTF8             bool
	ShowTypes        bool
	ShowFields       bool
	EpochDates       bool
	StreamRecordsMax int
}

func (f *WriterFlags) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&f.Format, "f", "zng", "format for output data [bzng,ndjson,table,text,types,zeek,zjson,zng]")
	fs.BoolVar(&f.ShowTypes, "T", false, "display field types in text output")
	fs.BoolVar(&f.ShowFields, "F", false, "display field names in text output")
	fs.BoolVar(&f.EpochDates, "E", false, "display epoch timestamps in text output")
	fs.BoolVar(&f.UTF8, "U", false, "display zeek strings as UTF-8")
	fs.IntVar(&f.StreamRecordsMax, "b", 0, "limit for number of records in each BZNG stream(0 for no limit)")
}

type Writer struct {
	zbuf.WriteFlusher
	io.Closer
}

func NewWriter(writer zbuf.WriteFlusher, closer io.Closer) *Writer {
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
	case "zng":
		return ".zng"
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
	case "bzng":
		return ".bzng"
	default:
		return ""
	}
}

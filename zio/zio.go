package zio

import (
	"flag"
	"io"

	"github.com/brimsec/zq/zbuf"
)

// ReaderFlags has the union of all the flags accepted by the different
// Reader implementations.
type ReaderFlags struct {
	Format   string
	ZngCheck bool
}

func (f *ReaderFlags) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&f.Format, "i", "auto", "format of input data [auto,zng,ndjson,zeek,zjson,tzng,parquet]")
	fs.BoolVar(&f.ZngCheck, "zngcheck", true, "check input records when reading ZNG streams")
}

// DefaultZngLZ4BlockSize is a reasonable default for
// WriterFlags.ZngLZ4BlockSize.
const DefaultZngLZ4BlockSize = 16 * 1024

// WriterFlags has the union of all the flags accepted by the different
// Writer implementations.
type WriterFlags struct {
	Format           string
	UTF8             bool
	ShowTypes        bool
	ShowFields       bool
	EpochDates       bool
	StreamRecordsMax int
	ZngLZ4BlockSize  int
}

func (f *WriterFlags) SetFlags(fs *flag.FlagSet) {
	fs.StringVar(&f.Format, "f", "zng", "format for output data [zng,ndjson,table,text,types,zeek,zjson,tzng]")
	fs.BoolVar(&f.ShowTypes, "T", false, "display field types in text output")
	fs.BoolVar(&f.ShowFields, "F", false, "display field names in text output")
	fs.BoolVar(&f.EpochDates, "E", false, "display epoch timestamps in text output")
	fs.BoolVar(&f.UTF8, "U", false, "display zeek strings as UTF-8")
	fs.IntVar(&f.StreamRecordsMax, "b", 0, "limit for number of records in each ZNG stream (0 for no limit)")
	fs.IntVar(&f.ZngLZ4BlockSize, "znglz4blocksize", DefaultZngLZ4BlockSize,
		"LZ4 block size in bytes for ZNG compression (nonpositive to disable)")
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
	case "tzng":
		return ".tzng"
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
	case "zng":
		return ".zng"
	default:
		return ""
	}
}

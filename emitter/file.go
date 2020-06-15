package emitter

import (
	"io"
	"os"

	"github.com/brimsec/zq/pkg/bufwriter"
	"github.com/brimsec/zq/pkg/iosource"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
)

type noClose struct {
	io.Writer
}

func (*noClose) Close() error {
	return nil
}

func NewFile(path string, flags *zio.WriterFlags) (*zio.Writer, error) {
	return NewFileWithSource(path, flags, iosource.DefaultRegistry)
}

func NewFileWithSource(path string, flags *zio.WriterFlags, source *iosource.Registry) (*zio.Writer, error) {
	var err error
	var f io.WriteCloser
	if path == "" {
		// Don't close stdout in case we live inside something
		// here that runs multiple instances of this to stdout.
		f = &noClose{os.Stdout}
	} else {
		f, err = source.NewWriter(path)
		if err != nil {
			return nil, err
		}
	}
	// On close, zio.Writer.Close(), the zng WriteFlusher will be flushed
	// then the bufwriter will closed (which will flush it's internal buffer
	// then close the file)
	w := detector.LookupWriter(bufwriter.New(f), flags)
	if w == nil {
		return nil, unknownFormat(flags.Format)
	}
	return w, nil
}

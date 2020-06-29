package emitter

import (
	"io"
	"os"

	"github.com/brimsec/zq/pkg/bufwriter"
	"github.com/brimsec/zq/pkg/iosrc"
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
	if path == "" {
		path = "stdout"
	}
	uri, err := iosrc.ParseURI(path)
	if err != nil {
		return nil, err
	}
	src, err := iosrc.GetSource(uri)
	if err != nil {
		return nil, err
	}
	return NewFileWithSource(uri, flags, src)
}

func NewFileWithSource(path iosrc.URI, flags *zio.WriterFlags, source iosrc.Source) (*zio.Writer, error) {
	f, err := source.NewWriter(path)
	if err != nil {
		return nil, err
	}
	if path.Scheme == "stdio" {
		// Don't close stdout in case we live inside something
		// here that runs multiple instances of this to stdout.
		f = &noClose{os.Stdout}
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

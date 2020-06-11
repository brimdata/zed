package emitter

import (
	"io"
	"os"

	"github.com/brimsec/zq/pkg/bufwriter"
	"github.com/brimsec/zq/pkg/fs"
	"github.com/brimsec/zq/pkg/s3io"
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
	var err error
	var f io.WriteCloser
	if path == "" {
		// Don't close stdout in case we live inside something
		// here that runs multiple instances of this to stdout.
		f = &noClose{os.Stdout}
	} else if s3io.IsS3Path(path) {
		if f, err = s3io.NewWriter(path, nil); err != nil {
			return nil, err
		}
	} else {
		flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		file, err := fs.OpenFile(path, flags, 0600)
		if err != nil {
			return nil, err
		}
		f = file
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

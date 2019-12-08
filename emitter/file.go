package emitter

import (
	"io"
	"os"

	"github.com/mccanne/zq/pkg/bufwriter"
	"github.com/mccanne/zq/pkg/zio"
	"github.com/mccanne/zq/pkg/zio/textio"
)

type noClose struct {
	io.Writer
}

func (p *noClose) Close() error {
	return nil
}

func NewFile(path, format string, tc *textio.Config) (*zio.Writer, error) {
	var f io.WriteCloser
	if path == "" {
		// Don't close stdout in case we live inside something
		// here that runs multiple instances of this to stdout.
		f = &noClose{os.Stdout}
	} else {
		var err error
		flags := os.O_WRONLY | os.O_CREATE
		file, err := os.OpenFile(path, flags, 0600)
		if err != nil {
			return nil, err
		}
		f = file
	}
	// On close, zio.Writer.Close(), the zson WriteFlusher will be flushed
	// then the bufwriter will closed (which will flush it's internal buffer
	// then close the file)
	w := zio.LookupWriter(format, bufwriter.New(f), tc)
	if w == nil {
		return nil, unknownFormat(format)
	}
	return w, nil
}

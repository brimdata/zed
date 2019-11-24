package emitter

import (
	"fmt"
	"io"
	"os"

	"github.com/mccanne/zq/pkg/bufwriter"
	"github.com/mccanne/zq/pkg/zsio"
)

type noClose struct {
	io.WriteCloser
}

func (p *noClose) Close() error {
	return nil
}

func OpenOutputFile(format, path string) (*zsio.Writer, error) {
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
	// On close, zsio.Writer.Close(), the zson WriteFlusher will be flushed
	// then the bufwriter will closed (which will flush it's internal buffer
	// then close the file)
	w := zsio.LookupWriter(format, bufwriter.New(f))
	if w == nil {
		return nil, fmt.Errorf("no such format: %s", format)
	}
	return w, nil
}

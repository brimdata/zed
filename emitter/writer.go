package emitter

import (
	"fmt"
	"io"
	"os"

	"github.com/mccanne/zq/pkg/bufwriter"
	"github.com/mccanne/zq/pkg/zsio"
	"github.com/mccanne/zq/pkg/zson"
)

type Writer interface {
	zson.Writer
	io.Closer
}

func OpenOutputFile(format, path string) (*zson.WriteCloser, error) {
	file := os.Stdout
	if path != "" {
		var err error
		flags := os.O_WRONLY | os.O_CREATE
		file, err = os.OpenFile(path, flags, 0600)
		if err != nil {
			return nil, err
		}
	}
	// Wrap the file (buffered by bufwriter) in a zeek writer,
	// then wrap that writer in zson.WriterCloser.  On close, the
	// zson.WriteCloser will call the bufwriter close, to flush it
	// and close the underlying file.
	w := bufwriter.New(file)
	zw := zsio.LookupWriter(format, w)
	if zw == nil {
		return nil, fmt.Errorf("no such format: %s", format)
	}
	return zson.NewWriteCloser(zw, w), nil
}

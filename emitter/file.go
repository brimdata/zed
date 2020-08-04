package emitter

import (
	"io"
	"os"

	"github.com/brimsec/zq/pkg/bufwriter"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
	"golang.org/x/crypto/ssh/terminal"
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

func isTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		if terminal.IsTerminal(int(f.Fd())) {
			return true
		}
	}
	return false
}

func NewFileWithSource(path iosrc.URI, flags *zio.WriterFlags, source iosrc.Source) (*zio.Writer, error) {
	f, err := source.NewWriter(path)
	if err != nil {
		return nil, err
	}

	var wc io.WriteCloser
	if path.Scheme == "stdio" {
		// Don't close stdio in case we live inside something
		// that has multiple stdio users.
		wc = &noClose{f}
		if !isTerminal(f) {
			// Don't buffer terminal output.
			wc = bufwriter.New(wc)
		}
	} else {
		wc = bufwriter.New(f)
	}
	// On close, zio.Writer.Close(), the zng WriteFlusher will be flushed
	// then the bufwriter will closed (which will flush it's internal buffer
	// then close the file)
	w := detector.LookupWriter(wc, flags)
	if w == nil {
		return nil, unknownFormat(flags.Format)
	}
	return w, nil
}

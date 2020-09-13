package emitter

import (
	"io"
	"os"

	"github.com/brimsec/zq/pkg/bufwriter"
	"github.com/brimsec/zq/pkg/iosrc"
	"github.com/brimsec/zq/zbuf"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/detector"
	"github.com/brimsec/zq/zio/options"
	"golang.org/x/crypto/ssh/terminal"
)

func NewFile(path string, opts options.Writer) (zbuf.WriteCloser, error) {
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
	return NewFileWithSource(uri, opts, src)
}

func IsTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		if terminal.IsTerminal(int(f.Fd())) {
			return true
		}
	}
	return false
}

func NewFileWithSource(path iosrc.URI, opts options.Writer, source iosrc.Source) (zbuf.WriteCloser, error) {
	f, err := source.NewWriter(path)
	if err != nil {
		return nil, err
	}

	var wc io.WriteCloser
	if path.Scheme == "stdio" {
		// Don't close stdio in case we live inside something
		// that has multiple stdio users.
		wc = zio.NopCloser(f)
		// Don't buffer terminal output.
		if !IsTerminal(f) {
			wc = bufwriter.New(wc)
		}
	} else {
		wc = bufwriter.New(f)
	}
	// On close, zbuf.WriteCloser.Close() will close and flush the
	// downstream writer, which will flush the bufwriter here and,
	// in turn, close its underlying writer.
	w := detector.LookupWriter(wc, opts)
	if w == nil {
		return nil, unknownFormat(opts.Format)
	}
	return w, nil
}

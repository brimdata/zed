package emitter

import (
	"context"
	"io"
	"os"

	"github.com/brimdata/zed/pkg/bufwriter"
	"github.com/brimdata/zed/pkg/iosrc"
	"github.com/brimdata/zed/pkg/terminal"
	"github.com/brimdata/zed/zbuf"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/anyio"
)

func NewFile(ctx context.Context, path string, opts anyio.WriterOpts) (zbuf.WriteCloser, error) {
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
	return NewFileWithSource(ctx, uri, opts, src)
}

func IsTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return terminal.IsTerminalFile(f)
	}
	return false
}

func NewFileWithSource(ctx context.Context, path iosrc.URI, opts anyio.WriterOpts, source iosrc.Source) (zbuf.WriteCloser, error) {
	f, err := source.NewWriter(ctx, path)
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
	w, err := anyio.LookupWriter(wc, opts)
	if err != nil {
		return nil, err
	}
	return w, nil
}

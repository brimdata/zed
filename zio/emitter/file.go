package emitter

import (
	"context"
	"io"
	"os"

	"github.com/brimdata/zed/pkg/bufwriter"
	"github.com/brimdata/zed/pkg/storage"
	"github.com/brimdata/zed/pkg/terminal"
	"github.com/brimdata/zed/zio"
	"github.com/brimdata/zed/zio/anyio"
)

func NewFileFromPath(ctx context.Context, engine storage.Engine, path string, opts anyio.WriterOpts) (zio.WriteCloser, error) {
	if path == "" {
		path = "stdio:stdout"
	}
	uri, err := storage.ParseURI(path)
	if err != nil {
		return nil, err
	}
	return NewFileFromURI(ctx, engine, uri, opts)
}

func IsTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return terminal.IsTerminalFile(f)
	}
	return false
}

func NewFileFromURI(ctx context.Context, engine storage.Engine, path *storage.URI, opts anyio.WriterOpts) (zio.WriteCloser, error) {
	f, err := engine.Put(ctx, path)
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
	// On close, zio.WriteCloser.Close will close and flush the
	// downstream writer, which will flush the bufwriter here and,
	// in turn, close its underlying writer.
	return anyio.NewWriter(wc, opts)
}

package azngio

import (
	"context"
	"io"
	"sync"

	atzngio "github.com/brimsec/zq/alpha/zio/tzngio"
	azngio "github.com/brimsec/zq/alpha/zio/zngio"
	aresolver "github.com/brimsec/zq/alpha/zng/resolver"
	"github.com/brimsec/zq/zio"
	"github.com/brimsec/zq/zio/tzngio"
	"github.com/brimsec/zq/zng"
	"github.com/brimsec/zq/zng/resolver"
)

// Reader provides a way to read the old alpha format of zng files and return
// them as an updated zng stream.  We do this by referring to the old alpha
// code in zq/alpha, creating an old zng reader that writes to an old tzng writer,
// and a new tzng reader that reads the old tzng using an io.Pipe.  Since tzng
// hasn't changed (except for "byte" changing to "uint8" and additions like
// end-of-stream) and since byte is not really used in practice (at least yet),
// it serves as the neutral translation format from alpha zng to beta zng.
type Reader struct {
	ar     *azngio.Reader
	zr     *tzngio.Reader
	once   sync.Once
	pipe   *io.PipeWriter
	ctx    context.Context
	cancel context.CancelFunc
}

func NewReader(r io.Reader, zctx *resolver.Context) *Reader {
	pipe, writer := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	return &Reader{
		ar:     azngio.NewReader(r, aresolver.NewContext()),
		zr:     tzngio.NewReader(pipe, zctx),
		pipe:   writer,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (r *Reader) Read() (*zng.Record, error) {
	r.once.Do(func() {
		go run(r.ctx, r.ar, r.pipe)
	})
	return r.zr.Read()
}

func (r *Reader) Close() error {
	// Shut down the thread by canceling and draining its write buffer.
	r.cancel()
	for {
		rec, err := r.zr.Read()
		if rec == nil || err != nil {
			if err == context.Canceled {
				err = nil
			}
			return err
		}
	}
}

func run(ctx context.Context, r *azngio.Reader, w *io.PipeWriter) {
	writer := atzngio.NewWriter(zio.NopCloser(w))
	for ctx.Err() == nil {
		rec, err := r.Read()
		if err != nil {
			writer.Close()
			w.CloseWithError(err)
			return
		}
		if rec == nil {
			if err := writer.Close(); err != nil {
				w.CloseWithError(err)
			} else {
				w.Close()
			}
			return
		}
		if err = writer.Write(rec); err != nil {
			writer.Close()
			w.CloseWithError(err)
			return
		}
	}
	w.CloseWithError(ctx.Err())
}
